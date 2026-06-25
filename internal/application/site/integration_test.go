package site_test

import (
	"context"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/prorochestvo/whoami/internal/application/site"
	"github.com/prorochestvo/whoami/internal/domain"
	resumerepo "github.com/prorochestvo/whoami/internal/repository/resume"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// nilFetcher is a stats fetcher that always returns an unavailable snapshot.
type nilFetcher struct{}

func (nilFetcher) Fetch(context.Context) (domain.GitHubStats, error) {
	return domain.GitHubStats{Available: false}, nil
}

// buildIntegration runs a real build using the embedded JSON repository and
// returns the output directory.
func buildIntegration(t *testing.T, locales []string, siteURL string) string {
	t.Helper()

	dir := t.TempDir()
	tmplDir := filepath.Join(dir, "templates")
	webDir := filepath.Join(dir, "web")
	outDir := filepath.Join(dir, "build")
	require.NoError(t, os.MkdirAll(tmplDir, 0o755))
	require.NoError(t, os.MkdirAll(webDir, 0o755))

	// minimal template that outputs hreflang and lang so we can assert on them
	tmpl := `{{define "layout"}}<html lang="{{.Lang}}">{{range .Alternates}}<link rel="alternate" hreflang="{{.Lang}}" href="{{.Href}}">{{end}}<title>{{.Title}}</title></html>{{end}}`
	require.NoError(t, os.WriteFile(filepath.Join(tmplDir, "layout.html.tmpl"), []byte(tmpl), 0o644))

	r, err := site.NewRenderer(tmplDir, webDir, outDir)
	require.NoError(t, err)

	logger := log.New(io.Discard, "", 0)
	b := site.NewBuilder(resumerepo.NewAdapter(), nilFetcher{}, r, locales, siteURL, logger)
	require.NoError(t, b.Build(context.Background()))
	return outDir
}

func TestIntegration_SingleLocale(t *testing.T) {
	t.Parallel()

	outDir := buildIntegration(t, []string{"en"}, "https://example.test/")

	t.Run("index.html exists at root", func(t *testing.T) {
		t.Parallel()
		_, err := os.Stat(filepath.Join(outDir, "index.html"))
		require.NoError(t, err)
	})

	t.Run("no build/en directory created", func(t *testing.T) {
		t.Parallel()
		_, err := os.Stat(filepath.Join(outDir, "en"))
		assert.True(t, os.IsNotExist(err))
	})

	t.Run("content/en.json emitted", func(t *testing.T) {
		t.Parallel()
		b, err := os.ReadFile(filepath.Join(outDir, "content", "en.json"))
		require.NoError(t, err)
		assert.Contains(t, string(b), `"locale"`)
		expected, err := resumerepo.Raw("en")
		require.NoError(t, err)
		assert.Equal(t, expected, b, "emitted JSON must be byte-identical to embedded source")
	})

	t.Run("sitemap.xml is valid and lists root URL", func(t *testing.T) {
		t.Parallel()
		b, err := os.ReadFile(filepath.Join(outDir, "sitemap.xml"))
		require.NoError(t, err)
		s := string(b)
		assert.Contains(t, s, "https://example.test/")
		assert.Contains(t, s, "x-default")
	})

	t.Run("html lang attribute is en", func(t *testing.T) {
		t.Parallel()
		b, err := os.ReadFile(filepath.Join(outDir, "index.html"))
		require.NoError(t, err)
		assert.Contains(t, string(b), `lang="en"`)
	})
}

func TestIntegration_MultiLocale(t *testing.T) {
	t.Parallel()

	outDir := buildIntegration(t, []string{"en", "qa"}, "https://example.test/")

	t.Run("default locale at root", func(t *testing.T) {
		t.Parallel()
		b, err := os.ReadFile(filepath.Join(outDir, "index.html"))
		require.NoError(t, err)
		assert.Contains(t, string(b), `lang="en"`)
	})

	t.Run("qa locale in subdirectory", func(t *testing.T) {
		t.Parallel()
		b, err := os.ReadFile(filepath.Join(outDir, "qa", "index.html"))
		require.NoError(t, err)
		assert.Contains(t, string(b), `lang="qa"`)
	})

	t.Run("content/qa.json emitted", func(t *testing.T) {
		t.Parallel()
		b, err := os.ReadFile(filepath.Join(outDir, "content", "qa.json"))
		require.NoError(t, err)
		expected, err := resumerepo.Raw("qa")
		require.NoError(t, err)
		assert.Equal(t, expected, b)
	})

	t.Run("en page contains hreflang for both locales and x-default", func(t *testing.T) {
		t.Parallel()
		b, err := os.ReadFile(filepath.Join(outDir, "index.html"))
		require.NoError(t, err)
		s := string(b)
		assert.Contains(t, s, `hreflang="en"`)
		assert.Contains(t, s, `hreflang="qa"`)
		assert.Contains(t, s, `hreflang="x-default"`)
		// x-default must point at the default (root) URL
		assert.Contains(t, s, `hreflang="x-default" href="https://example.test/"`)
	})

	t.Run("qa page contains hreflang for both locales and x-default", func(t *testing.T) {
		t.Parallel()
		b, err := os.ReadFile(filepath.Join(outDir, "qa", "index.html"))
		require.NoError(t, err)
		s := string(b)
		assert.Contains(t, s, `hreflang="en"`)
		assert.Contains(t, s, `hreflang="qa"`)
		assert.Contains(t, s, `hreflang="x-default"`)
	})

	t.Run("sitemap lists both locale URLs", func(t *testing.T) {
		t.Parallel()
		b, err := os.ReadFile(filepath.Join(outDir, "sitemap.xml"))
		require.NoError(t, err)
		s := string(b)
		assert.Contains(t, s, "https://example.test/")
		assert.Contains(t, s, "https://example.test/qa/")
		assert.Contains(t, s, "x-default")
	})

	t.Run("qa locale content is visibly marked as placeholder", func(t *testing.T) {
		t.Parallel()
		b, err := os.ReadFile(filepath.Join(outDir, "content", "qa.json"))
		require.NoError(t, err)
		s := string(b)
		assert.True(t,
			strings.Contains(s, "PLACEHOLDER") || strings.Contains(s, "placeholder"),
			"qa locale content must contain PLACEHOLDER marker",
		)
	})
}
