package site

import (
	"context"
	"errors"
	"io"
	"log"
	"os"
	"path/filepath"
	"testing"

	"github.com/prorochestvo/whoami/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var _ StatsFetcher = (*fakeFetcher)(nil)
var _ ContentSource = (*fakeContent)(nil)
var _ PDFRenderer = (*fakePDF)(nil)

// sentinelPDF is the fixed byte string the fake PDFRenderer returns so tests can
// assert exactly which bytes landed at each per-locale path.
var sentinelPDF = []byte("%PDF-1.7\nsentinel-cv\n%%EOF\n")

// fakePDF is a stub PDFRenderer returning fixed bytes or a fixed error.
type fakePDF struct {
	out []byte
	err error
}

func (f *fakePDF) Render(domain.Resume) ([]byte, error) {
	return f.out, f.err
}

type fakeFetcher struct {
	stats domain.GitHubStats
	err   error
}

func (f fakeFetcher) Fetch(context.Context) (domain.GitHubStats, error) {
	return f.stats, f.err
}

// fakeContent provides two locales: "en" and optionally a second one.
type fakeContent struct {
	locales map[string]domain.Resume
	rawJSON map[string][]byte
}

func newFakeContent(locales ...string) *fakeContent {
	fc := &fakeContent{
		locales: make(map[string]domain.Resume),
		rawJSON: make(map[string][]byte),
	}
	for _, l := range locales {
		fc.locales[l] = domain.Resume{Person: domain.Person{Name: "Test " + l}}
		fc.rawJSON[l] = []byte(`{"locale":"` + l + `","localeName":"Name ` + l + `"}`)
	}
	return fc
}

func (f *fakeContent) Available() []string {
	keys := make([]string, 0, len(f.locales))
	for k := range f.locales {
		keys = append(keys, k)
	}
	return keys
}

func (f *fakeContent) Load(locale string) (domain.Resume, error) {
	r, ok := f.locales[locale]
	if !ok {
		return domain.Resume{}, errors.New("no such locale: " + locale)
	}
	return r, nil
}

func (f *fakeContent) Raw(locale string) ([]byte, error) {
	b, ok := f.rawJSON[locale]
	if !ok {
		return nil, errors.New("no raw for locale: " + locale)
	}
	return b, nil
}

func newTestRenderer(t *testing.T) (*Renderer, string) {
	t.Helper()
	return newTestRendererWith(t, &fakePDF{out: sentinelPDF})
}

func newTestRendererWith(t *testing.T, pdf PDFRenderer) (*Renderer, string) {
	t.Helper()
	dir := t.TempDir()
	tmplDir := filepath.Join(dir, "templates")
	webDir := filepath.Join(dir, "web")
	outDir := filepath.Join(dir, "build")
	require.NoError(t, os.MkdirAll(tmplDir, 0o755))
	require.NoError(t, os.MkdirAll(webDir, 0o755))
	tmpl := `{{define "layout"}}{{.Resume.Person.Name}}|{{.Lang}}|{{if .GitHub.Available}}STATS:{{.GitHub.PublicRepos}}{{else}}NOSTATS{{end}}{{end}}`
	require.NoError(t, os.WriteFile(filepath.Join(tmplDir, "layout.html.tmpl"), []byte(tmpl), 0o644))

	r, err := NewRenderer(tmplDir, webDir, outDir, pdf)
	require.NoError(t, err)
	return r, outDir
}

func TestBuilder_Build(t *testing.T) {
	t.Parallel()

	logger := log.New(io.Discard, "", 0)

	t.Run("bakes stats when the fetch succeeds", func(t *testing.T) {
		t.Parallel()
		r, outDir := newTestRenderer(t)
		content := newFakeContent("en")
		f := fakeFetcher{stats: domain.GitHubStats{Available: true, PublicRepos: 7}}

		err := NewBuilder(content, f, r, []string{"en"}, "https://example.test/", logger).Build(context.Background())
		require.NoError(t, err)

		html, err := os.ReadFile(filepath.Join(outDir, "index.html"))
		require.NoError(t, err)
		assert.Contains(t, string(html), "Test en|en|STATS:7")
	})

	t.Run("renders without stats and no error when the fetch fails", func(t *testing.T) {
		t.Parallel()
		r, outDir := newTestRenderer(t)
		content := newFakeContent("en")
		f := fakeFetcher{err: errors.New("offline")}

		err := NewBuilder(content, f, r, []string{"en"}, "https://example.test/", logger).Build(context.Background())
		require.NoError(t, err)

		html, err := os.ReadFile(filepath.Join(outDir, "index.html"))
		require.NoError(t, err)
		assert.Contains(t, string(html), "Test en|en|NOSTATS")
	})

	t.Run("renders multiple locales with fetch failure gracefully", func(t *testing.T) {
		t.Parallel()
		r, outDir := newTestRenderer(t)
		content := newFakeContent("en", "qa")
		f := fakeFetcher{err: errors.New("offline")}

		err := NewBuilder(content, f, r, []string{"en", "qa"}, "https://example.test/", logger).Build(context.Background())
		require.NoError(t, err)

		// default locale at root
		html, err := os.ReadFile(filepath.Join(outDir, "index.html"))
		require.NoError(t, err)
		assert.Contains(t, string(html), "Test en|en|NOSTATS")

		// non-default locale in subdirectory
		htmlQA, err := os.ReadFile(filepath.Join(outDir, "qa", "index.html"))
		require.NoError(t, err)
		assert.Contains(t, string(htmlQA), "Test qa|qa|NOSTATS")
	})

	t.Run("emits locale JSON files", func(t *testing.T) {
		t.Parallel()
		r, outDir := newTestRenderer(t)
		content := newFakeContent("en")
		f := fakeFetcher{}

		err := NewBuilder(content, f, r, []string{"en"}, "https://example.test/", logger).Build(context.Background())
		require.NoError(t, err)

		jsonBytes, err := os.ReadFile(filepath.Join(outDir, "content", "en.json"))
		require.NoError(t, err)
		assert.Equal(t, content.rawJSON["en"], jsonBytes, "emitted JSON must be byte-identical to embedded source")
	})

	t.Run("emits sitemap.xml listing all locale URLs", func(t *testing.T) {
		t.Parallel()
		r, outDir := newTestRenderer(t)
		content := newFakeContent("en", "qa")
		f := fakeFetcher{}

		err := NewBuilder(content, f, r, []string{"en", "qa"}, "https://example.test/", logger).Build(context.Background())
		require.NoError(t, err)

		sitemap, err := os.ReadFile(filepath.Join(outDir, "sitemap.xml"))
		require.NoError(t, err)
		s := string(sitemap)
		assert.Contains(t, s, "https://example.test/", "sitemap must list default locale URL")
		assert.Contains(t, s, "https://example.test/qa/", "sitemap must list qa locale URL")
		assert.Contains(t, s, "x-default", "sitemap must include x-default hreflang")
	})

	t.Run("assets copied once regardless of locale count", func(t *testing.T) {
		t.Parallel()
		r, outDir := newTestRenderer(t)
		content := newFakeContent("en", "qa")
		f := fakeFetcher{}

		// put a sentinel file in webDir to verify it gets copied
		webDir := filepath.Join(filepath.Dir(outDir), "web")
		require.NoError(t, os.WriteFile(filepath.Join(webDir, "sentinel.txt"), []byte("ok"), 0o644))

		err := NewBuilder(content, f, r, []string{"en", "qa"}, "https://example.test/", logger).Build(context.Background())
		require.NoError(t, err)

		// sentinel must be in outDir (assets copied exactly once)
		_, err = os.Stat(filepath.Join(outDir, "sentinel.txt"))
		require.NoError(t, err, "sentinel asset must be present in output")
	})

	t.Run("writes cv.pdf per locale via the PDF renderer", func(t *testing.T) {
		t.Parallel()
		r, outDir := newTestRenderer(t)
		content := newFakeContent("en", "qa")
		f := fakeFetcher{}

		err := NewBuilder(content, f, r, []string{"en", "qa"}, "https://example.test/", logger).Build(context.Background())
		require.NoError(t, err)

		en, err := os.ReadFile(filepath.Join(outDir, "cv.pdf"))
		require.NoError(t, err)
		assert.Equal(t, sentinelPDF, en, "en cv.pdf lands at the root with the sentinel bytes")

		qa, err := os.ReadFile(filepath.Join(outDir, "qa", "cv.pdf"))
		require.NoError(t, err)
		assert.Equal(t, sentinelPDF, qa, "qa cv.pdf lands under its locale dir")

		_, err = os.Stat(filepath.Join(outDir, "en", "cv.pdf"))
		assert.True(t, os.IsNotExist(err), "no build/en/cv.pdf for the default locale")
	})

	t.Run("no build/en directory created for default locale", func(t *testing.T) {
		t.Parallel()
		r, outDir := newTestRenderer(t)
		content := newFakeContent("en")
		f := fakeFetcher{}

		err := NewBuilder(content, f, r, []string{"en"}, "https://example.test/", logger).Build(context.Background())
		require.NoError(t, err)

		_, err = os.Stat(filepath.Join(outDir, "en"))
		assert.True(t, os.IsNotExist(err), "build/en/ must not be created for the default locale")
	})
}
