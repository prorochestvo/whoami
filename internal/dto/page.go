// Package dto holds the view models passed to the HTML templates. They flatten
// the domain aggregates into exactly what a template needs, including SEO fields.
package dto

import (
	"encoding/json"
	"html/template"
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

	OGImage       string
	OGImageAlt    string
	OGImageWidth  int
	OGImageHeight int

	// OGLocale is this page's OpenGraph locale ("" when the code is unmapped);
	// OGLocaleAlternates holds the OG locales of the other mapped, built locales.
	OGLocale           string
	OGLocaleAlternates []string

	// PersonLD is a ready-to-emit <script type="application/ld+json"> block of
	// schema.org Person data, built by buildPersonLD. It is template.HTML because
	// the payload is first-party résumé data already HTML-escaped by json.Marshal;
	// see buildPersonLD for the injection-safety rationale.
	PersonLD template.HTML

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

	// PersonLD is a progressive enhancement: json.Marshal of a struct of strings
	// cannot fail in practice (invalid UTF-8 is replaced, not errored), and the
	// block's absence must never fail the build — so a marshal error degrades to an
	// empty block rather than forcing a (Page, error) signature onto every caller.
	personImage := siteURL + "img/avatar/site-transparent-512.png"
	personLD, err := buildPersonLD(r.Person.Name, r.Person.Title, canonical, personImage, profileURLs(r.Contacts))
	if err != nil {
		personLD = ""
	}

	return Page{
		Resume:       r,
		GitHub:       s,
		Year:         generatedAt.Year(),
		GeneratedAt:  generatedAt,
		Title:        r.Person.Name + " — " + r.Person.Title,
		Description:  r.Person.Tagline,
		CanonicalURL: canonical,

		OGImage:       siteURL + "img/og.png",
		OGImageAlt:    r.Person.Name + " — " + r.Person.Title,
		OGImageWidth:  ogImageWidth,
		OGImageHeight: ogImageHeight,

		OGLocale:           ogLocale(lang),
		OGLocaleAlternates: ogLocaleAlternates(lang, allLocales),
		PersonLD:           personLD,

		Lang:        lang,
		LocaleName:  localeName(allLocales, lang),
		Alternates:  alternates,
		LocaleLinks: links,

		PDFHref:         pdfHref(lang),
		PDFDownloadName: pdfDownloadName(r.Person.Name, lang),
	}
}

// ogImageWidth and ogImageHeight are the fixed pixel dimensions of the committed
// og.png card; naming them here keeps every og:image:* value in one place.
const (
	ogImageWidth  = 1200
	ogImageHeight = 630
)

// ogLocaleByLang maps a content locale code to its OpenGraph locale. A code with
// no entry (the qa test placeholder, or any future unmapped locale) yields "",
// so the page omits its own og:locale and is skipped as an alternate elsewhere.
var ogLocaleByLang = map[string]string{
	"en": "en_US",
	"ru": "ru_RU",
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

// ogLocale returns the OpenGraph locale for a content code, or "" when the code
// has no mapping.
func ogLocale(lang string) string {
	return ogLocaleByLang[lang]
}

// ogLocaleAlternates returns the OG locales of every built locale other than
// current, in order, skipping the current locale and any unmapped code so a page
// never emits an og:locale:alternate for itself or for an unmapped placeholder.
func ogLocaleAlternates(current string, all []LocaleMeta) []string {
	var out []string
	for _, m := range all {
		if m.Code == current {
			continue
		}
		if loc := ogLocale(m.Code); loc != "" {
			out = append(out, loc)
		}
	}
	return out
}

// profileURLs collects the absolute http(s) contact URLs in order, for the
// schema.org Person sameAs list. Keeping the derivation in the data (rather than
// hardcoding profile links) drops mailto: and URL-less contacts and makes adding
// a profile a content-only change.
func profileURLs(contacts []domain.Contact) []string {
	var out []string
	for _, c := range contacts {
		if strings.HasPrefix(c.URL, "https://") || strings.HasPrefix(c.URL, "http://") {
			out = append(out, c.URL)
		}
	}
	return out
}

// buildPersonLD renders a schema.org Person as a ready-to-emit ld+json script.
// Injection safety rests on two independent guards: (1) json.Marshal escapes '<',
// '>' and '&' to U+003C, U+003E and U+0026 (and U+2028/U+2029), so no value can
// emit a literal "</script>" that terminates the element; (2) the <script>
// wrapper is a static literal, never assembled from résumé data. The result is
// template.HTML because the payload is first-party data already HTML-escaped by
// json.Marshal — not raw external input — so this is not a bypass on external
// data. Do not switch to a json.Encoder with SetEscapeHTML(false): that removes
// guard #1. image and url must be absolute, as schema.org requires.
func buildPersonLD(name, jobTitle, url, image string, sameAs []string) (template.HTML, error) {
	doc := struct {
		Context  string   `json:"@context"`
		Type     string   `json:"@type"`
		Name     string   `json:"name"`
		JobTitle string   `json:"jobTitle"`
		URL      string   `json:"url"`
		Image    string   `json:"image"`
		SameAs   []string `json:"sameAs,omitempty"`
	}{
		Context:  "https://schema.org",
		Type:     "Person",
		Name:     name,
		JobTitle: jobTitle,
		URL:      url,
		Image:    image,
		SameAs:   sameAs,
	}
	b, err := json.Marshal(doc)
	if err != nil {
		return "", err
	}
	return template.HTML(`<script type="application/ld+json">` + string(b) + `</script>`), nil
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
