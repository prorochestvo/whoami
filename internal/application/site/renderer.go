package site

import (
	"encoding/xml"
	"fmt"
	"html/template"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/prorochestvo/whoami/internal/dto"
)

func NewRenderer(templatesDir, webDir, outDir string) (*Renderer, error) {
	tmpl, err := template.New("site").Funcs(funcMap()).ParseGlob(filepath.Join(templatesDir, "*.tmpl"))
	if err != nil {
		return nil, fmt.Errorf("render: parse templates: %w", err)
	}
	return &Renderer{tmpl: tmpl, webDir: webDir, outDir: outDir}, nil
}

type Renderer struct {
	tmpl   *template.Template
	webDir string
	outDir string
}

// Render writes the page to its per-locale output path.
func (r *Renderer) Render(page dto.Page) error {
	if err := os.MkdirAll(r.outDir, 0o755); err != nil {
		return fmt.Errorf("render: create out dir: %w", err)
	}
	return r.writeLocale(page)
}

// CopyAssets mirrors the static asset tree from webDir into outDir.
func (r *Renderer) CopyAssets() error {
	return filepath.WalkDir(r.webDir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(r.webDir, path)
		if err != nil {
			return err
		}
		dst := filepath.Join(r.outDir, rel)
		if d.IsDir() {
			return os.MkdirAll(dst, 0o755)
		}
		return copyFile(path, dst)
	})
}

// WriteLocaleJSON writes the raw locale JSON to outDir/content/<locale>.json for
// the client-side switcher. The bytes must be the embedded source verbatim.
func (r *Renderer) WriteLocaleJSON(locale string, raw []byte) error {
	dir := filepath.Join(r.outDir, "content")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("render: create content dir: %w", err)
	}
	dst := filepath.Join(dir, locale+".json")
	if err := os.WriteFile(dst, raw, 0o644); err != nil {
		return fmt.Errorf("render: write %s.json: %w", locale, err)
	}
	return nil
}

// WriteSitemap writes outDir/sitemap.xml from the page view models. allPages must
// be in locale order; every page shares the same hreflang alternate set.
func (r *Renderer) WriteSitemap(allPages []dto.Page, siteURL string) error {
	if !strings.HasSuffix(siteURL, "/") {
		siteURL += "/"
	}

	type xLink struct {
		XMLName  xml.Name `xml:"http://www.w3.org/1999/xhtml link"`
		Rel      string   `xml:"rel,attr"`
		HrefLang string   `xml:"hreflang,attr"`
		Href     string   `xml:"href,attr"`
	}
	type urlEntry struct {
		XMLName xml.Name `xml:"url"`
		Loc     string   `xml:"loc"`
		Links   []xLink  `xml:"http://www.w3.org/1999/xhtml link"`
	}
	type urlSet struct {
		XMLName xml.Name   `xml:"urlset"`
		XMLNS   string     `xml:"xmlns,attr"`
		XSI     string     `xml:"xmlns:xsi,attr"`
		Schema  string     `xml:"xsi:schemaLocation,attr"`
		XHTML   string     `xml:"xmlns:xhtml,attr"`
		URLs    []urlEntry `xml:"url"`
	}

	urls := make([]urlEntry, len(allPages))
	for i, p := range allPages {
		xlinks := make([]xLink, len(p.Alternates))
		for j, a := range p.Alternates {
			xlinks[j] = xLink{Rel: "alternate", HrefLang: a.Lang, Href: a.Href}
		}
		urls[i] = urlEntry{Loc: p.CanonicalURL, Links: xlinks}
	}

	doc := urlSet{
		XMLNS:  "http://www.sitemaps.org/schemas/sitemap/0.9",
		XSI:    "http://www.w3.org/2001/XMLSchema-instance",
		Schema: "http://www.sitemaps.org/schemas/sitemap/0.9 http://www.sitemaps.org/schemas/sitemap/0.9/sitemap.xsd",
		XHTML:  "http://www.w3.org/1999/xhtml",
		URLs:   urls,
	}

	out, err := xml.MarshalIndent(doc, "", "  ")
	if err != nil {
		return fmt.Errorf("render: marshal sitemap: %w", err)
	}

	dst := filepath.Join(r.outDir, "sitemap.xml")
	content := []byte(xml.Header + string(out) + "\n")
	if err := os.WriteFile(dst, content, 0o644); err != nil {
		return fmt.Errorf("render: write sitemap.xml: %w", err)
	}
	return nil
}

// writeLocale renders the layout to the per-locale path: "en" → outDir/index.html,
// others → outDir/<lang>/index.html.
func (r *Renderer) writeLocale(page dto.Page) error {
	var dir string
	if page.Lang == "en" {
		dir = r.outDir
	} else {
		dir = filepath.Join(r.outDir, page.Lang)
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return fmt.Errorf("render: create locale dir %s: %w", page.Lang, err)
		}
	}

	f, err := os.Create(filepath.Join(dir, "index.html"))
	if err != nil {
		return fmt.Errorf("render: create index.html for %s: %w", page.Lang, err)
	}
	defer func() { _ = f.Close() }()

	if err := r.tmpl.ExecuteTemplate(f, "layout", page); err != nil {
		return fmt.Errorf("render: execute template for %s: %w", page.Lang, err)
	}
	return nil
}

// copyFile copies src to dst, creating parents. The dst Close error is surfaced
// via the named return so a flush failure is not swallowed.
func copyFile(src, dst string) (retErr error) {
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer func() { _ = in.Close() }()

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer func() {
		if cerr := out.Close(); cerr != nil && retErr == nil {
			retErr = fmt.Errorf("render: close %s: %w", dst, cerr)
		}
	}()

	if _, err := io.Copy(out, in); err != nil {
		return fmt.Errorf("render: copy %s: %w", src, err)
	}
	return nil
}

func funcMap() template.FuncMap {
	return template.FuncMap{
		"join":  strings.Join,
		"lower": strings.ToLower,
	}
}
