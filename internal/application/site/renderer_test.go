package site

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/prorochestvo/whoami/internal/domain"
	"github.com/prorochestvo/whoami/internal/dto"
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
