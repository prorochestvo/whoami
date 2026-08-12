package site

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/prorochestvo/whoami/internal/domain"
	"github.com/prorochestvo/whoami/internal/dto"
	resumerepo "github.com/prorochestvo/whoami/internal/repository/resume"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRenderer_WritePDF(t *testing.T) {
	t.Parallel()

	resume := domain.Resume{Person: domain.Person{Name: "Test"}}

	t.Run("writes the en PDF at the output root", func(t *testing.T) {
		t.Parallel()
		r, outDir := newTestRendererWith(t, &fakePDF{out: sentinelPDF})
		require.NoError(t, r.WritePDF("en", resume))

		b, err := os.ReadFile(filepath.Join(outDir, "cv.pdf"))
		require.NoError(t, err)
		assert.Equal(t, sentinelPDF, b)
	})

	t.Run("writes a non-en PDF under its locale directory", func(t *testing.T) {
		t.Parallel()
		r, outDir := newTestRendererWith(t, &fakePDF{out: sentinelPDF})
		require.NoError(t, r.WritePDF("ru", resume))

		b, err := os.ReadFile(filepath.Join(outDir, "ru", "cv.pdf"))
		require.NoError(t, err)
		assert.Equal(t, sentinelPDF, b)

		_, err = os.Stat(filepath.Join(outDir, "cv.pdf"))
		assert.True(t, os.IsNotExist(err), "a non-en locale must not write the root cv.pdf")
	})

	t.Run("wraps and surfaces a generator error without writing a file", func(t *testing.T) {
		t.Parallel()
		genErr := errors.New("boom")
		r, outDir := newTestRendererWith(t, &fakePDF{err: genErr})

		err := r.WritePDF("en", resume)
		require.Error(t, err)
		assert.ErrorIs(t, err, genErr, "the generator error must be wrapped, not swallowed")

		_, statErr := os.Stat(filepath.Join(outDir, "cv.pdf"))
		assert.True(t, os.IsNotExist(statErr), "no file may be written when generation fails")
	})
}

// TestRenderer_Render_downloadLink renders the real production template (not the
// synthetic one used elsewhere) to confirm the download anchor is wired to the
// view model's PDFHref and PDFDownloadName per locale, and survives with no JS.
func TestRenderer_Render_downloadLink(t *testing.T) {
	t.Parallel()

	// Locate the repo-root templates dir from this test file's own path so the
	// test does not depend on the working directory.
	_, thisFile, _, ok := runtime.Caller(0)
	require.True(t, ok)
	templatesDir := filepath.Join(filepath.Dir(thisFile), "..", "..", "..", "templates")

	dir := t.TempDir()
	webDir := filepath.Join(dir, "web")
	outDir := filepath.Join(dir, "build")
	require.NoError(t, os.MkdirAll(webDir, 0o755))
	r, err := NewRenderer(templatesDir, webDir, outDir, &fakePDF{out: sentinelPDF})
	require.NoError(t, err)

	resume := domain.Resume{Person: domain.Person{
		Name: "Seilbek Skindirov", Title: "Engineer", Tagline: "builds things",
		Experience: "17+ years", GitHubUser: "prorochestvo",
	}}
	locales := []dto.LocaleMeta{{Code: "en", Name: "English"}, {Code: "ru", Name: "Русский"}}
	at := time.Date(2026, 8, 5, 0, 0, 0, 0, time.UTC)

	t.Run("en page links the root CV with an ASCII download name", func(t *testing.T) {
		page := dto.NewPage(resume, domain.GitHubStats{}, "https://example.test/", "en", locales, at)
		require.NoError(t, r.Render(page))
		html, err := os.ReadFile(filepath.Join(outDir, "index.html"))
		require.NoError(t, err)
		s := string(html)
		assert.Contains(t, s, `class="download-cv-icon"`)
		assert.Contains(t, s, `href="/cv.pdf"`)
		assert.Contains(t, s, `download="Seilbek-Skindirov-CV.pdf"`)
		assert.Contains(t, s, `aria-label="Download CV (PDF)"`, "the icon link must carry an accessible name")
	})

	t.Run("ru page links the ru CV per locale", func(t *testing.T) {
		page := dto.NewPage(resume, domain.GitHubStats{}, "https://example.test/", "ru", locales, at)
		require.NoError(t, r.Render(page))
		html, err := os.ReadFile(filepath.Join(outDir, "ru", "index.html"))
		require.NoError(t, err)
		s := string(html)
		assert.Contains(t, s, `href="/ru/cv.pdf"`)
		assert.Contains(t, s, `download="Seilbek-Skindirov-CV-ru.pdf"`)
	})
}

// TestRenderer_Render_headMetadata renders the real production layout with the
// real embedded résumés to confirm the SEO/link-preview head tags and the
// schema.org Person JSON-LD are wired from the view model, per locale.
func TestRenderer_Render_headMetadata(t *testing.T) {
	t.Parallel()

	_, thisFile, _, ok := runtime.Caller(0)
	require.True(t, ok)
	templatesDir := filepath.Join(filepath.Dir(thisFile), "..", "..", "..", "templates")

	dir := t.TempDir()
	webDir := filepath.Join(dir, "web")
	outDir := filepath.Join(dir, "build")
	require.NoError(t, os.MkdirAll(webDir, 0o755))
	r, err := NewRenderer(templatesDir, webDir, outDir, &fakePDF{out: sentinelPDF})
	require.NoError(t, err)

	enResume, err := resumerepo.Load("en")
	require.NoError(t, err)
	ruResume, err := resumerepo.Load("ru")
	require.NoError(t, err)

	locales := []dto.LocaleMeta{{Code: "en", Name: "English"}, {Code: "ru", Name: "Русский"}}
	at := time.Date(2026, 8, 6, 0, 0, 0, 0, time.UTC)

	t.Run("en head carries OG, Twitter, icon and JSON-LD tags", func(t *testing.T) {
		page := dto.NewPage(enResume, domain.GitHubStats{}, "https://example.test/", "en", locales, at)
		require.NoError(t, r.Render(page))
		s := readFileString(t, filepath.Join(outDir, "index.html"))

		assert.Contains(t, s, `property="og:image:alt"`)
		assert.Contains(t, s, `property="og:image:width" content="1200"`)
		assert.Contains(t, s, `property="og:image:height" content="630"`)
		assert.Contains(t, s, `property="og:site_name" content="Seilbek Skindirov"`)
		assert.Contains(t, s, `name="twitter:image" content="https://example.test/img/og.png"`)
		assert.Contains(t, s, `property="og:locale" content="en_US"`)
		assert.Contains(t, s, `property="og:locale:alternate" content="ru_RU"`)

		assert.Contains(t, s, `<link rel="icon" href="/img/favicon-32.png?v=`)
		assert.Contains(t, s, `<link rel="icon" href="/img/favicon-16.png?v=`)
		assert.Contains(t, s, `<link rel="apple-touch-icon" href="/img/apple-touch-icon.png?v=`)
		assert.NotContains(t, s, "favicon.svg", "the terminal favicon must be gone")

		// sameAs is GitHub + Telegram + LinkedIn — hh is intentionally omitted.
		ld := extractLD(t, s)
		var doc map[string]any
		require.NoError(t, json.Unmarshal([]byte(ld), &doc))
		assert.Equal(t, []any{
			"https://github.com/prorochestvo",
			"https://t.me/prorochestvo",
			"https://www.linkedin.com/in/prorochestvo/",
		}, doc["sameAs"])
	})

	t.Run("ru head carries ru_RU locale and a parseable Cyrillic JSON-LD", func(t *testing.T) {
		page := dto.NewPage(ruResume, domain.GitHubStats{}, "https://example.test/", "ru", locales, at)
		require.NoError(t, r.Render(page))
		s := readFileString(t, filepath.Join(outDir, "ru", "index.html"))

		assert.Contains(t, s, `property="og:locale" content="ru_RU"`)
		assert.Contains(t, s, `property="og:locale:alternate" content="en_US"`)

		ld := extractLD(t, s)
		var doc map[string]any
		require.NoError(t, json.Unmarshal([]byte(ld), &doc))
		assert.Equal(t, "Сейльбек Скиндиров", doc["name"], "the Cyrillic name must survive escaping")
		assert.Equal(t, "Разработчик ПО · Full-Stack инженер", doc["jobTitle"])
		assert.Equal(t, "https://example.test/ru/", doc["url"])
		assert.Equal(t, "https://example.test/img/avatar/site-transparent-512.png", doc["image"])
	})
}

// extractLD returns the JSON payload of the single ld+json block. json.Marshal
// escapes any "</script>" in the data, so the first literal </script> after the
// opening tag is always the block's own close.
func extractLD(t *testing.T, html string) string {
	t.Helper()
	const open = `<script type="application/ld+json">`
	const closeTag = `</script>`
	i := strings.Index(html, open)
	require.GreaterOrEqual(t, i, 0, "an ld+json block must be present")
	rest := html[i+len(open):]
	j := strings.Index(rest, closeTag)
	require.GreaterOrEqual(t, j, 0, "the ld+json block must close")
	return rest[:j]
}

func readFileString(t *testing.T, path string) string {
	t.Helper()
	b, err := os.ReadFile(path)
	require.NoError(t, err)
	return string(b)
}
