// Package dto holds the view models passed to the HTML templates. They flatten
// the domain aggregates into exactly what a template needs, including SEO fields.
package dto

import (
	"strings"
	"time"
	"unicode"

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

	// PDFHref is the root-relative URL of the per-locale CV PDF ("/cv.pdf" for
	// en, "/<lang>/cv.pdf" otherwise), matching how the template references other
	// root-relative assets. PDFDownloadName is the ASCII save-name for the
	// anchor's download attribute.
	PDFHref         string
	PDFDownloadName string
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

		PDFHref:         pdfHref(lang),
		PDFDownloadName: pdfDownloadName(r.Person.Name, lang),
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

// pdfHref returns the root-relative CV PDF URL for a locale, mirroring the HTML
// convention: "en" at the root, every other locale under its own path prefix.
func pdfHref(lang string) string {
	if lang == "en" {
		return "/cv.pdf"
	}
	return "/" + lang + "/cv.pdf"
}

// pdfDownloadName derives the anchor's ASCII save-name from the person's name:
// an ASCII slug suffixed "-CV" (and "-<lang>" for non-en). A name that strips to
// nothing (e.g. the Cyrillic ru name) falls back to "CV-<lang>.pdf" so the
// download attribute never carries a non-ASCII, cross-platform-unsafe filename.
func pdfDownloadName(name, lang string) string {
	slug := asciiSlug(name)
	if slug == "" {
		return "CV-" + lang + ".pdf"
	}
	if lang == "en" {
		return slug + "-CV.pdf"
	}
	return slug + "-CV-" + lang + ".pdf"
}

// asciiSlug keeps ASCII letters and digits, treats whitespace as a token break
// joined by single hyphens, and drops every other rune (punctuation, non-ASCII)
// without splitting. It returns "" when nothing ASCII-alphanumeric survives.
func asciiSlug(s string) string {
	var tokens []string
	var cur strings.Builder
	flush := func() {
		if cur.Len() > 0 {
			tokens = append(tokens, cur.String())
			cur.Reset()
		}
	}
	for _, r := range s {
		switch {
		case (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9'):
			cur.WriteRune(r)
		case unicode.IsSpace(r):
			flush()
		}
	}
	flush()
	return strings.Join(tokens, "-")
}
