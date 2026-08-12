package dto

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/prorochestvo/whoami/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewPage(t *testing.T) {
	t.Parallel()

	resume := domain.Resume{Person: domain.Person{
		Name:    "Jane Doe",
		Title:   "Engineer",
		Tagline: "builds things",
	}}
	at := time.Date(2026, 6, 20, 0, 0, 0, 0, time.UTC)
	singleLocale := []LocaleMeta{{Code: "en", Name: "English"}}
	twoLocales := []LocaleMeta{{Code: "en", Name: "English"}, {Code: "de", Name: "Deutsch"}}

	t.Run("derives SEO fields from the résumé", func(t *testing.T) {
		t.Parallel()
		p := NewPage(resume, domain.GitHubStats{}, "https://example.test/", "en", singleLocale, at)
		assert.Equal(t, "Jane Doe — Engineer", p.Title)
		assert.Equal(t, "builds things", p.Description)
		assert.Equal(t, 2026, p.Year)
	})

	t.Run("normalizes a missing trailing slash so the OG image URL is valid", func(t *testing.T) {
		t.Parallel()
		p := NewPage(resume, domain.GitHubStats{}, "https://example.test", "en", singleLocale, at)
		assert.Equal(t, "https://example.test/", p.CanonicalURL)
		assert.Equal(t, "https://example.test/img/og.png", p.OGImage)
	})

	t.Run("default locale canonical has no prefix", func(t *testing.T) {
		t.Parallel()
		p := NewPage(resume, domain.GitHubStats{}, "https://example.test/", "en", singleLocale, at)
		assert.Equal(t, "https://example.test/", p.CanonicalURL)
		assert.Equal(t, "en", p.Lang)
		assert.Equal(t, "English", p.LocaleName)
	})

	t.Run("non-default locale canonical is prefixed", func(t *testing.T) {
		t.Parallel()
		p := NewPage(resume, domain.GitHubStats{}, "https://example.test/", "de", twoLocales, at)
		assert.Equal(t, "https://example.test/de/", p.CanonicalURL)
		assert.Equal(t, "de", p.Lang)
		assert.Equal(t, "Deutsch", p.LocaleName)
	})

	t.Run("alternates include x-default pointing at default", func(t *testing.T) {
		t.Parallel()
		p := NewPage(resume, domain.GitHubStats{}, "https://example.test/", "de", twoLocales, at)
		require.Len(t, p.Alternates, 3, "en + de + x-default")

		var xdef, en, de *Alternate
		for i := range p.Alternates {
			a := &p.Alternates[i]
			switch a.Lang {
			case "x-default":
				xdef = a
			case "en":
				en = a
			case "de":
				de = a
			}
		}
		require.NotNil(t, xdef, "x-default must be present")
		require.NotNil(t, en, "en alternate must be present")
		require.NotNil(t, de, "de alternate must be present")

		assert.Equal(t, "https://example.test/", xdef.Href, "x-default points at default locale")
		assert.Equal(t, "https://example.test/", en.Href)
		assert.Equal(t, "https://example.test/de/", de.Href)
	})

	t.Run("single locale alternates contain en and x-default", func(t *testing.T) {
		t.Parallel()
		p := NewPage(resume, domain.GitHubStats{}, "https://example.test/", "en", singleLocale, at)
		require.Len(t, p.Alternates, 2, "en + x-default")
	})

	t.Run("locale links mark current correctly", func(t *testing.T) {
		t.Parallel()
		p := NewPage(resume, domain.GitHubStats{}, "https://example.test/", "de", twoLocales, at)
		require.Len(t, p.LocaleLinks, 2)
		for _, l := range p.LocaleLinks {
			if l.Lang == "de" {
				assert.True(t, l.Current)
			} else {
				assert.False(t, l.Current)
			}
		}
	})

	t.Run("PDF href is root-relative and per-locale", func(t *testing.T) {
		t.Parallel()
		en := NewPage(resume, domain.GitHubStats{}, "https://example.test/", "en", singleLocale, at)
		assert.Equal(t, "/cv.pdf", en.PDFHref)
		de := NewPage(resume, domain.GitHubStats{}, "https://example.test/", "de", twoLocales, at)
		assert.Equal(t, "/de/cv.pdf", de.PDFHref)
	})

	t.Run("PDF download name is an ASCII slug suffixed per locale", func(t *testing.T) {
		t.Parallel()
		latin := domain.Resume{Person: domain.Person{Name: "Seilbek Skindirov"}}
		en := NewPage(latin, domain.GitHubStats{}, "https://example.test/", "en", singleLocale, at)
		assert.Equal(t, "Seilbek-Skindirov-CV.pdf", en.PDFDownloadName)
		de := NewPage(latin, domain.GitHubStats{}, "https://example.test/", "de", twoLocales, at)
		assert.Equal(t, "Seilbek-Skindirov-CV-de.pdf", de.PDFDownloadName)
	})

	t.Run("PDF download name falls back to ASCII for a non-ASCII name", func(t *testing.T) {
		t.Parallel()
		cyrillic := domain.Resume{Person: domain.Person{Name: "Сейльбек Скиндиров"}}
		locales := []LocaleMeta{{Code: "en", Name: "English"}, {Code: "ru", Name: "Русский"}}
		p := NewPage(cyrillic, domain.GitHubStats{}, "https://example.test/", "ru", locales, at)
		assert.Equal(t, "/ru/cv.pdf", p.PDFHref)
		assert.Equal(t, "CV-ru.pdf", p.PDFDownloadName)
	})

	t.Run("OG image dimensions and alt describe the shared card", func(t *testing.T) {
		t.Parallel()
		p := NewPage(resume, domain.GitHubStats{}, "https://example.test/", "en", singleLocale, at)
		assert.Equal(t, 1200, p.OGImageWidth)
		assert.Equal(t, 630, p.OGImageHeight)
		assert.Equal(t, resume.Person.Name+" — "+resume.Person.Title, p.OGImageAlt)
	})

	t.Run("OG locale maps per page and excludes the current and unmapped from alternates", func(t *testing.T) {
		t.Parallel()
		locales := []LocaleMeta{{Code: "en", Name: "English"}, {Code: "ru", Name: "Русский"}, {Code: "qa", Name: "QA"}}

		en := NewPage(resume, domain.GitHubStats{}, "https://example.test/", "en", locales, at)
		assert.Equal(t, "en_US", en.OGLocale)
		assert.Equal(t, []string{"ru_RU"}, en.OGLocaleAlternates, "qa is unmapped and must be skipped")

		ru := NewPage(resume, domain.GitHubStats{}, "https://example.test/", "ru", locales, at)
		assert.Equal(t, "ru_RU", ru.OGLocale)
		assert.Equal(t, []string{"en_US"}, ru.OGLocaleAlternates)

		qa := NewPage(resume, domain.GitHubStats{}, "https://example.test/", "qa", locales, at)
		assert.Equal(t, "", qa.OGLocale, "qa has no OG mapping")
		assert.Equal(t, []string{"en_US", "ru_RU"}, qa.OGLocaleAlternates)
	})

	t.Run("an unmapped current locale yields an empty OG locale but still lists mapped alternates", func(t *testing.T) {
		t.Parallel()
		p := NewPage(resume, domain.GitHubStats{}, "https://example.test/", "de", twoLocales, at)
		assert.Equal(t, "", p.OGLocale, "de is unmapped")
		assert.Equal(t, []string{"en_US"}, p.OGLocaleAlternates, "en still maps even when the current locale does not")
	})

	t.Run("populates a parseable Person JSON-LD with a localized job title and derived sameAs", func(t *testing.T) {
		t.Parallel()
		r := domain.Resume{
			Person: domain.Person{Name: "Сейльбек Скиндиров", Title: "Разработчик ПО"},
			Contacts: []domain.Contact{
				{Label: "Email", URL: "mailto:a@b.c"},
				{Label: "GitHub", URL: "https://github.com/prorochestvo"},
				{Label: "Telegram", URL: "https://t.me/prorochestvo"},
				{Label: "LinkedIn", URL: "https://www.linkedin.com/in/prorochestvo/"},
			},
		}
		p := NewPage(r, domain.GitHubStats{}, "https://example.test/", "ru", []LocaleMeta{{Code: "ru", Name: "Русский"}}, at)

		s := string(p.PersonLD)
		require.True(t, strings.HasPrefix(s, `<script type="application/ld+json">`))
		inner := strings.TrimSuffix(strings.TrimPrefix(s, `<script type="application/ld+json">`), `</script>`)

		var doc map[string]any
		require.NoError(t, json.Unmarshal([]byte(inner), &doc))
		assert.Equal(t, "Сейльбек Скиндиров", doc["name"], "the Cyrillic name round-trips cleanly")
		assert.Equal(t, "Разработчик ПО", doc["jobTitle"], "jobTitle comes from the loaded locale")
		assert.Equal(t, "https://example.test/img/avatar/site-transparent-512.png", doc["image"], "image is absolute")
		assert.Equal(t, "https://example.test/ru/", doc["url"], "url is the per-locale canonical")
		assert.Equal(t, []any{
			"https://github.com/prorochestvo",
			"https://t.me/prorochestvo",
			"https://www.linkedin.com/in/prorochestvo/",
		}, doc["sameAs"], "sameAs is GitHub + Telegram + LinkedIn — no mailto, no hh")
	})
}

func TestProfileURLs(t *testing.T) {
	t.Parallel()

	t.Run("keeps only absolute http(s) URLs in order", func(t *testing.T) {
		t.Parallel()
		contacts := []domain.Contact{
			{Label: "Location", Value: "Kazakhstan", URL: ""},
			{Label: "Email", Value: "a@b.c", URL: "mailto:a@b.c"},
			{Label: "GitHub", Value: "github.com/x", URL: "https://github.com/x"},
			{Label: "Telegram", Value: "@x", URL: "https://t.me/x"},
		}
		assert.Equal(t, []string{"https://github.com/x", "https://t.me/x"}, profileURLs(contacts))
	})

	t.Run("returns nothing for contactless input", func(t *testing.T) {
		t.Parallel()
		assert.Empty(t, profileURLs(nil))
	})
}

func TestBuildPersonLD(t *testing.T) {
	t.Parallel()

	const (
		wrapPre  = `<script type="application/ld+json">`
		wrapPost = `</script>`
	)
	// inner strips the static wrapper to expose the marshaled JSON payload.
	inner := func(t *testing.T, s string) string {
		t.Helper()
		require.True(t, strings.HasPrefix(s, wrapPre), "must open with the ld+json script tag")
		require.True(t, strings.HasSuffix(s, wrapPost), "must close with </script>")
		return strings.TrimSuffix(strings.TrimPrefix(s, wrapPre), wrapPost)
	}

	t.Run("emits a valid ld+json object with the expected fields", func(t *testing.T) {
		t.Parallel()
		block, err := buildPersonLD(
			"Jane Doe", "Engineer",
			"https://example.test/", "https://example.test/img/avatar/site-transparent-512.png",
			[]string{"https://github.com/x", "https://t.me/x"},
		)
		require.NoError(t, err)

		var doc map[string]any
		require.NoError(t, json.Unmarshal([]byte(inner(t, string(block))), &doc))
		assert.Equal(t, "https://schema.org", doc["@context"])
		assert.Equal(t, "Person", doc["@type"])
		assert.Equal(t, "Jane Doe", doc["name"])
		assert.Equal(t, "Engineer", doc["jobTitle"])
		assert.Equal(t, "https://example.test/", doc["url"])
		assert.Equal(t, "https://example.test/img/avatar/site-transparent-512.png", doc["image"])
		assert.Equal(t, []any{"https://github.com/x", "https://t.me/x"}, doc["sameAs"])
	})

	t.Run("escapes a hostile name so it cannot break out of the script element", func(t *testing.T) {
		t.Parallel()
		const hostile = `Seilbek </script><script>alert(1)</script>`
		block, err := buildPersonLD(hostile, "Engineer", "https://example.test/", "https://example.test/img.png", nil)
		require.NoError(t, err)

		s := string(block)
		assert.NotContains(t, s, `</script><script>alert`, "the raw break-out sequence must not survive")

		// Derive the escaped closing tag from encoding/json itself rather than
		// hardcoding the escape sequence: the payload must carry only that form.
		escClose, err := json.Marshal("</script>")
		require.NoError(t, err)
		assert.Contains(t, s, strings.Trim(string(escClose), `"`), "the closing tag survives only escaped")

		// The escaping must be reversible, not lossy: the name round-trips back to
		// the exact hostile literal after unmarshaling.
		var doc map[string]any
		require.NoError(t, json.Unmarshal([]byte(inner(t, s)), &doc))
		assert.Equal(t, hostile, doc["name"])
	})

	t.Run("omits sameAs when empty", func(t *testing.T) {
		t.Parallel()
		block, err := buildPersonLD("Jane Doe", "Engineer", "https://example.test/", "https://example.test/img.png", nil)
		require.NoError(t, err)
		assert.NotContains(t, inner(t, string(block)), `"sameAs"`, "omitempty must drop an empty sameAs")
	})
}
