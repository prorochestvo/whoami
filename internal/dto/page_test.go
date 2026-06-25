package dto

import (
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
}
