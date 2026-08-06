package pdf

import (
	"bytes"
	"encoding/binary"
	"testing"
	"unicode/utf16"

	"github.com/prorochestvo/whoami/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// englishResume is a compact but structurally complete Latin résumé used across
// the black-box subtests. The final project omits repo, stack and highlights and
// the second experience omits arrangement — the optional fields pdf.go skips when
// empty (it never reads URL fields; the CV is plain text by design).
func englishResume() domain.Resume {
	return domain.Resume{
		Person: domain.Person{
			Name:       "Seilbek Skindirov",
			Title:      "Software Developer",
			Location:   "Kazakhstan",
			Experience: "17+ years",
			Tagline:    "Backends, integrations and full-stack products.",
		},
		About: []string{"Full-stack and backend engineer.", "Comfortable across the stack."},
		Experience: []domain.Experience{
			{
				Role: "Backend Engineer", Company: "Repfabric", Period: "Apr 2021 — Present",
				Location: "Tahoma, US", Arrangement: "Remote · Full-time",
				Summary:    "Automation and synchronization tools.",
				Highlights: []string{"Built CRM/ERP integration entry points.", "Expanded test coverage."},
				Stack:      []string{"Scala", "ZIO", "Dart"}, Current: true,
			},
			{
				Role: "Contract Developer", Company: "Space Management LTD", Period: "Jul 2014 — Feb 2015",
				Location: "Astana", Arrangement: "", // empty arrangement: no dangling separator
				Summary:    "Access-control hardware integration.",
				Highlights: []string{"Connected turnstiles to a backend server."},
				Stack:      []string{"C++"},
			},
		},
		Projects: []domain.Project{
			{
				Name: "Beacon", Repo: "github.com/prorochestvo/beacon", URL: "https://github.com/prorochestvo/beacon",
				Stack: []string{"Go", "SQLite"}, Summary: "FX-rate Telegram bot.",
				Highlights: []string{"Multi-binary architecture."},
			},
			{Name: "WhoAmI", Summary: "Static-site CV generator."}, // no repo, url, stack or highlights
		},
		Education: []domain.Education{
			{Year: "2008", Institution: "Kostanay STU", Field: "Computer Engineering", Degree: "Bachelor"},
		},
		Skills: []domain.SkillGroup{
			{Name: "Languages", Items: []string{"Go", "Scala", "PHP"}},
			{Name: "Infra & Tools", Items: []string{"Docker", "Git", "Nginx"}},
		},
		Languages: []domain.Language{
			{Name: "Russian", Level: "Native"},
			{Name: "English", Level: "Elementary (A2)"},
		},
		Contacts: []domain.Contact{
			{Label: "Email", Value: "SeilbekSkindirov@gmail.com", URL: "mailto:SeilbekSkindirov@gmail.com"},
			{Label: "GitHub", Value: "github.com/prorochestvo", URL: "https://github.com/prorochestvo"},
		},
	}
}

// cyrillicResume mirrors englishResume with Cyrillic content in every text field
// so a font/encoding regression (glyphs rendered as boxes) surfaces here.
func cyrillicResume() domain.Resume {
	return domain.Resume{
		Person: domain.Person{
			Name:       "Сейльбек Скиндиров",
			Title:      "Разработчик ПО",
			Location:   "Казахстан",
			Experience: "17+ лет",
			Tagline:    "Бэкенды, интеграции и full-stack продукты.",
		},
		About: []string{"Full-stack и backend инженер."},
		Experience: []domain.Experience{{
			Role: "Backend-инженер", Company: "Repfabric", Period: "Апр 2021 — наст. время",
			Location: "Тахома", Arrangement: "Удалённо · Полная занятость",
			Summary:    "Инструменты автоматизации бизнес-процессов.",
			Highlights: []string{"Спроектировал точки входа для CRM/ERP."},
			Stack:      []string{"Scala", "ZIO"}, Current: true,
		}},
		Projects:  []domain.Project{{Name: "Beacon", Summary: "Боевой Telegram-бот."}},
		Education: []domain.Education{{Year: "2008", Institution: "КСТУ им. З. Алдамжара", Field: "ВТиПО", Degree: "Бакалавр"}},
		Skills:    []domain.SkillGroup{{Name: "Languages", Items: []string{"Go", "Scala"}}},
		Languages: []domain.Language{{Name: "Русский", Level: "Родной"}},
		Contacts:  []domain.Contact{{Label: "Email", Value: "SeilbekSkindirov@gmail.com"}},
	}
}

func TestRenderer_Render(t *testing.T) {
	t.Parallel()

	t.Run("produces a valid, non-trivial PDF", func(t *testing.T) {
		t.Parallel()
		out, err := New().Render(englishResume())
		require.NoError(t, err)
		assert.True(t, bytes.HasPrefix(out, []byte("%PDF-")), "must start with the %%PDF- magic")
		assert.True(t, bytes.Contains(out, []byte("%%EOF")), "must contain the %%EOF trailer")
		assert.Greater(t, len(out), 3*1024, "an embedded-font PDF is well over 3 KB")
	})

	t.Run("renders the Cyrillic résumé without error", func(t *testing.T) {
		t.Parallel()
		out, err := New().Render(cyrillicResume())
		require.NoError(t, err)
		assert.True(t, bytes.HasPrefix(out, []byte("%PDF-")))
		assert.True(t, bytes.Contains(out, []byte("%%EOF")))
	})

	t.Run("is deterministic for identical input", func(t *testing.T) {
		t.Parallel()
		a, err := New().Render(cyrillicResume())
		require.NoError(t, err)
		b, err := New().Render(cyrillicResume())
		require.NoError(t, err)
		assert.Equal(t, a, b, "fixed creation date must keep bytes reproducible across runs")
	})

	// White-box seam: build() lets the test disable stream compression (kept out
	// of the public API) so the content stream is inspectable. fpdf writes text
	// for a UTF-8 font as UTF-16BE code points via a Type0/Identity-H composite
	// font, so the Cyrillic name and an English heading must appear as their
	// UTF-16BE byte sequences — proof the glyphs were laid down through the
	// embedded font (real glyphs), not dropped to .notdef boxes.
	t.Run("embeds the UTF-8 font and encodes Cyrillic glyphs", func(t *testing.T) {
		t.Parallel()
		p, err := build(cyrillicResume())
		require.NoError(t, err)
		p.SetCompression(false)
		var buf bytes.Buffer
		require.NoError(t, p.Output(&buf))
		raw := buf.Bytes()

		assert.Contains(t, string(raw), "/BaseFont /utf8jbmono",
			"the embedded UTF-8 font must be the active font, not a core font")
		assert.Contains(t, string(raw), "FontFile2", "the TTF must be embedded in the output")
		assert.True(t, bytes.Contains(raw, utf16be("Скиндиров")),
			"the Cyrillic name must be encoded in the content stream (not rendered as boxes)")
		assert.True(t, bytes.Contains(raw, utf16be(headingExperience)),
			"a section heading must be encoded in the content stream")
	})

	t.Run("renders empty optional fields cleanly", func(t *testing.T) {
		t.Parallel()
		// Only-empty-optionals: a project with no repo/stack/summary/highlights and
		// an experience with no period/location/arrangement/summary/highlights, so
		// every optional-skip branch in build() is exercised.
		res := domain.Resume{
			Person:     domain.Person{Name: "Minimal"},
			Experience: []domain.Experience{{Role: "Engineer"}},
			Projects:   []domain.Project{{Name: "Tool"}},
		}
		out, err := New().Render(res)
		require.NoError(t, err)
		assert.True(t, bytes.HasPrefix(out, []byte("%PDF-")))
		assert.True(t, bytes.Contains(out, []byte("%%EOF")))
	})

	// Leaf contract: fpdf accumulates a deferred error rather than faulting per
	// call, so Render must check it and return a wrapped pdf: error with no bytes
	// — never silent, malformed output. U+1F600 sits past the UTF-8 font's width
	// table, which trips that deferred state (fpdf: "character outside the
	// supported range").
	t.Run("surfaces fpdf's deferred error as a wrapped pdf error", func(t *testing.T) {
		t.Parallel()
		res := domain.Resume{
			Person:     domain.Person{Name: "Grinning \U0001F600"},
			Experience: []domain.Experience{{Role: "Engineer", Company: "ACME"}},
		}
		out, err := New().Render(res)
		require.Error(t, err)
		assert.Nil(t, out, "no bytes may be returned when generation fails")
		assert.Contains(t, err.Error(), "pdf:", "the fpdf error must be wrapped in the package's error prefix")
	})
}

// utf16be encodes s as big-endian UTF-16, matching how fpdf writes UTF-8 font
// text into the PDF content stream.
func utf16be(s string) []byte {
	u := utf16.Encode([]rune(s))
	b := make([]byte, 2*len(u))
	for i, v := range u {
		binary.BigEndian.PutUint16(b[i*2:], v)
	}
	return b
}
