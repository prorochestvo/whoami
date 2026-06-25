// Package dto holds the view models passed to the HTML templates. They flatten
// the domain aggregates into exactly what a template needs, including SEO fields.
package dto

import (
	"strings"
	"time"

	"github.com/prorochestvo/whoami/internal/domain"
)

type LocaleMeta struct {
	Code string
	Name string
}

// Alternate is one hreflang entry; Href must be absolute.
type Alternate struct {
	Lang string
	Href string
}

type LocaleLink struct {
	Lang    string
	Name    string
	Href    string
	Current bool
}

type Page struct {
	Resume      domain.Resume
	GitHub      domain.GitHubStats
	Year        int
	GeneratedAt time.Time

	Title        string
	Description  string
	CanonicalURL string
	OGImage      string

	Lang        string
	LocaleName  string
	Alternates  []Alternate
	LocaleLinks []LocaleLink
}

// NewPage builds the view model for one locale. siteURL's trailing slash is
// normalized; the "en" entry in allLocales becomes the x-default hreflang target.
func NewPage(r domain.Resume, s domain.GitHubStats, siteURL, lang string, allLocales []LocaleMeta, generatedAt time.Time) Page {
	if !strings.HasSuffix(siteURL, "/") {
		siteURL += "/"
	}

	canonical := localeURL(siteURL, lang)

	alternates := make([]Alternate, 0, len(allLocales)+1)
	for _, m := range allLocales {
		alternates = append(alternates, Alternate{
			Lang: m.Code,
			Href: localeURL(siteURL, m.Code),
		})
	}
	// x-default points at the default locale (en). The default locale always
	// renders at the bare siteURL with no path prefix.
	alternates = append(alternates, Alternate{Lang: "x-default", Href: siteURL})

	links := make([]LocaleLink, len(allLocales))
	for i, m := range allLocales {
		links[i] = LocaleLink{
			Lang:    m.Code,
			Name:    m.Name,
			Href:    localeURL(siteURL, m.Code),
			Current: m.Code == lang,
		}
	}

	return Page{
		Resume:       r,
		GitHub:       s,
		Year:         generatedAt.Year(),
		GeneratedAt:  generatedAt,
		Title:        r.Person.Name + " — " + r.Person.Title,
		Description:  r.Person.Tagline,
		CanonicalURL: canonical,
		OGImage:      siteURL + "img/og.png",
		Lang:         lang,
		LocaleName:   localeName(allLocales, lang),
		Alternates:   alternates,
		LocaleLinks:  links,
	}
}

// localeURL returns the absolute URL for a locale; "en" lives at the bare siteURL
// with no prefix so the indexed "/" stays stable.
func localeURL(siteURL, lang string) string {
	if lang == "en" {
		return siteURL
	}
	return siteURL + lang + "/"
}

// localeName returns the display name for code, falling back to the code itself.
func localeName(locales []LocaleMeta, code string) string {
	for _, m := range locales {
		if m.Code == code {
			return m.Name
		}
	}
	return code
}
