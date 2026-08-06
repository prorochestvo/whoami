// Package pdf renders a domain.Resume into a print- and ATS-friendly A4 PDF at
// build time. It is a pure "domain aggregate -> bytes" adapter over the go-pdf/fpdf
// library: no filesystem, no locale, and no output-path knowledge live here — the
// already-localized Resume carries every locale-varying string, so the renderer
// needs no locale parameter. JetBrains Mono (Regular + Bold) is embedded and
// registered as a UTF-8 font so Latin, Cyrillic and Greek all render as real,
// selectable glyphs. Generation performs no external call, so any failure is a
// programming error surfaced as a wrapped, non-nil error rather than a degraded
// document.
package pdf

import (
	"bytes"
	_ "embed"
	"fmt"
	"strings"
	"time"

	"github.com/go-pdf/fpdf"

	"github.com/prorochestvo/whoami/internal/domain"
)

// New returns a stateless PDF renderer. It mirrors github.NewClient: a constructor
// per infrastructure adapter, cheap to build and safe to reuse across locales.
func New() *Renderer { return &Renderer{} }

// Renderer turns a domain.Resume into PDF bytes. The zero value is ready to use;
// prefer New for symmetry with the other infrastructure adapters.
type Renderer struct{}

// Render builds the A4 CV document for r and returns the encoded PDF bytes. The
// bytes are deterministic for identical input (a fixed creation/modification date
// is stamped, never time.Now). It returns a wrapped, non-nil error if fpdf's
// deferred error state is set (for example a font that failed to register) or if
// encoding the output fails.
func (*Renderer) Render(r domain.Resume) ([]byte, error) {
	p, err := build(r)
	if err != nil {
		return nil, err
	}
	var buf bytes.Buffer
	if err := p.Output(&buf); err != nil {
		return nil, fmt.Errorf("pdf: output: %w", err)
	}
	return buf.Bytes(), nil
}

//go:embed fonts/JetBrainsMono-Regular.ttf
var fontRegular []byte

//go:embed fonts/JetBrainsMono-Bold.ttf
var fontBold []byte

// fontFamily is the internal family key both embedded styles register under.
const fontFamily = "JBMono"

// fixedEpoch keeps output bytes reproducible across rebuilds: fpdf stamps
// time.Now into the document's CreationDate/ModDate otherwise, which would churn
// the deployed file on every scheduled rebuild even when the CV is unchanged.
var fixedEpoch = time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC)

// Page geometry in millimetres.
const (
	marginX      = 18.0
	marginTop    = 16.0
	marginBottom = 16.0

	skillLabelW   = 32.0 // label column width for the skills section
	contactLabelW = 26.0 // label column width for the contacts section
)

// Section headings are English constants, mirroring the template's hardcoded
// "// experience" / "// projects" structural labels: the site uses English
// section labels across every locale, so the PDF does the same.
const (
	headingAbout      = "About"
	headingExperience = "Experience"
	headingProjects   = "Projects"
	headingEducation  = "Education"
	headingSkills     = "Skills"
	headingLanguages  = "Languages"
	headingContacts   = "Contacts"
)

// Palette: near-monochrome ink on white with a single dark-green accent echoing
// the site's brand green (#7ee787), darkened to #2E7D32 for contrast on paper.
var (
	ink    = rgb{26, 26, 26}
	dim    = rgb{90, 90, 90}
	accent = rgb{46, 125, 50}
	rule   = rgb{206, 206, 206}
)

// build assembles the fpdf document from r. It is an unexported seam so tests can
// disable stream compression and inspect the raw output; the public API never
// exposes the *fpdf.Fpdf. It returns fpdf's accumulated deferred error, if any.
func build(r domain.Resume) (*fpdf.Fpdf, error) {
	p := fpdf.New("P", "mm", "A4", "")
	// SetCatalogSort forces sorted (deterministic) PDF object emission: fpdf
	// otherwise numbers font objects in Go map-iteration order, which is
	// randomized and would churn the deployed file on every rebuild. Combined
	// with the fixed dates below, output is byte-reproducible for equal input.
	p.SetCatalogSort(true)
	p.SetCreationDate(fixedEpoch)
	p.SetModificationDate(fixedEpoch)
	p.AddUTF8FontFromBytes(fontFamily, "", fontRegular)
	p.AddUTF8FontFromBytes(fontFamily, "B", fontBold)
	p.SetMargins(marginX, marginTop, marginX)
	p.SetAutoPageBreak(true, marginBottom)
	if name := strings.TrimSpace(r.Person.Name); name != "" {
		p.SetTitle(name+" — CV", true)
	}
	p.AddPage()

	pageW, _ := p.GetPageSize()
	d := &document{p: p, cw: pageW - 2*marginX}
	d.header(r.Person)

	if len(r.About) > 0 {
		d.heading(headingAbout)
		for _, para := range r.About {
			d.line("", 9.5, ink, 4.6, para)
		}
	}
	if len(r.Experience) > 0 {
		d.heading(headingExperience)
		for _, e := range r.Experience {
			d.experience(e)
		}
	}
	if len(r.Projects) > 0 {
		d.heading(headingProjects)
		for _, pr := range r.Projects {
			d.project(pr)
		}
	}
	if len(r.Education) > 0 {
		d.heading(headingEducation)
		for _, e := range r.Education {
			d.education(e)
		}
	}
	if len(r.Skills) > 0 {
		d.heading(headingSkills)
		for _, s := range r.Skills {
			d.labeled(s.Name, strings.Join(s.Items, ", "), skillLabelW)
		}
	}
	if len(r.Languages) > 0 {
		d.heading(headingLanguages)
		parts := make([]string, 0, len(r.Languages))
		for _, l := range r.Languages {
			parts = append(parts, joinNonEmpty(" — ", l.Name, l.Level))
		}
		d.line("", 9.5, ink, 4.6, strings.Join(parts, " · "))
	}
	if len(r.Contacts) > 0 {
		d.heading(headingContacts)
		for _, c := range r.Contacts {
			d.labeled(c.Label, c.Value, contactLabelW)
		}
	}

	if err := p.Error(); err != nil {
		return nil, fmt.Errorf("pdf: build: %w", err)
	}
	return p, nil
}

// document wraps the fpdf handle with the usable content width so the section
// helpers stay short. All rendering flows left-aligned, top-to-bottom; fpdf's
// MultiCell handles word wrap and SetAutoPageBreak handles pagination.
type document struct {
	p  *fpdf.Fpdf
	cw float64
}

// education renders one degree: bold institution then a dimmed meta line.
func (d *document) education(e domain.Education) {
	d.p.SetFont(fontFamily, "B", 10.5)
	d.p.SetTextColor(ink.r, ink.g, ink.b)
	d.p.MultiCell(d.cw, 5.0, e.Institution, "", "L", false)
	if meta := joinNonEmpty(" · ", e.Degree, e.Field, e.Year); meta != "" {
		d.line("", 9, dim, 4.2, meta)
	}
	d.p.Ln(1.5)
}

// experience renders one position: bold "role @ company", a dimmed meta line,
// the summary, hanging-indent highlight bullets and a compact stack line.
func (d *document) experience(e domain.Experience) {
	title := e.Role
	if e.Company != "" {
		title += "  @ " + e.Company
	}
	d.line("B", 10.5, ink, 5.2, title)
	if meta := joinNonEmpty(" · ", e.Period, e.Location, e.Arrangement); meta != "" {
		d.line("", 9, dim, 4.2, meta)
	}
	if e.Summary != "" {
		d.line("", 9.5, ink, 4.6, e.Summary)
	}
	if len(e.Highlights) > 0 {
		d.bullets(e.Highlights)
	}
	if len(e.Stack) > 0 {
		d.line("", 8.8, dim, 4.2, strings.Join(e.Stack, " · "))
	}
	d.p.Ln(2.5)
}

// header renders the name (accent), title, a location/experience meta line and
// the tagline, followed by a full-width rule.
func (d *document) header(p domain.Person) {
	d.line("B", 20, accent, 8.5, p.Name)
	if p.Title != "" {
		d.line("", 10.5, dim, 5.0, p.Title)
	}
	if meta := joinNonEmpty(" · ", p.Location, p.Experience); meta != "" {
		d.line("", 9, dim, 4.2, meta)
	}
	if p.Tagline != "" {
		d.line("", 9.5, ink, 4.6, p.Tagline)
	}
	d.p.Ln(1.5)
	d.horizontalRule(0.4)
}

// heading renders a bold section title over a thin full-width rule.
func (d *document) heading(title string) {
	d.p.Ln(3)
	d.p.SetFont(fontFamily, "B", 12)
	d.p.SetTextColor(ink.r, ink.g, ink.b)
	d.p.MultiCell(d.cw, 6, title, "", "L", false)
	d.horizontalRule(0.2)
	d.p.Ln(1.5)
}

// project renders one portfolio entry. Repo, stack and highlights are optional
// (a private/NDA project has no URL; some entries have no highlights) and are
// simply skipped when empty.
func (d *document) project(p domain.Project) {
	d.line("B", 10.5, ink, 5.2, p.Name)
	if p.Repo != "" {
		d.line("", 8.8, dim, 4.2, p.Repo)
	}
	if len(p.Stack) > 0 {
		d.line("", 8.8, dim, 4.2, strings.Join(p.Stack, " · "))
	}
	if p.Summary != "" {
		d.line("", 9.5, ink, 4.6, p.Summary)
	}
	if len(p.Highlights) > 0 {
		d.bullets(p.Highlights)
	}
	d.p.Ln(2.5)
}

// bullets renders each item as an accent marker with a hanging indent: wrapped
// lines align under the text, not under the marker. The left margin is restored
// after every item so a mid-item page break keeps the indent.
func (d *document) bullets(items []string) {
	const indent = 4.5
	left, _, _, _ := d.p.GetMargins()
	for _, it := range items {
		d.p.SetXY(left, d.p.GetY())
		d.p.SetFont(fontFamily, "B", 9.5)
		d.p.SetTextColor(accent.r, accent.g, accent.b)
		d.p.CellFormat(indent, 4.6, "•", "", 0, "L", false, 0, "")
		d.p.SetLeftMargin(left + indent)
		d.p.SetX(left + indent)
		d.p.SetFont(fontFamily, "", 9.5)
		d.p.SetTextColor(ink.r, ink.g, ink.b)
		d.p.MultiCell(d.cw-indent, 4.6, it, "", "L", false)
		d.p.SetLeftMargin(left)
	}
}

// horizontalRule draws a full-width rule of the given thickness at the cursor.
func (d *document) horizontalRule(thickness float64) {
	left, _, _, _ := d.p.GetMargins()
	y := d.p.GetY()
	d.p.SetDrawColor(rule.r, rule.g, rule.b)
	d.p.SetLineWidth(thickness)
	d.p.Line(left, y, left+d.cw, y)
	d.p.Ln(1)
}

// labeled renders a bold label in a fixed-width column with the body wrapping in
// a hanging indent to its right, used for the skills and contacts rows.
func (d *document) labeled(label, body string, labelW float64) {
	left, _, _, _ := d.p.GetMargins()
	d.p.SetXY(left, d.p.GetY())
	d.p.SetFont(fontFamily, "B", 9.5)
	d.p.SetTextColor(ink.r, ink.g, ink.b)
	d.p.CellFormat(labelW, 4.8, label, "", 0, "L", false, 0, "")
	d.p.SetLeftMargin(left + labelW)
	d.p.SetX(left + labelW)
	d.p.SetFont(fontFamily, "", 9.5)
	d.p.SetTextColor(ink.r, ink.g, ink.b)
	d.p.MultiCell(d.cw-labelW, 4.8, body, "", "L", false)
	d.p.SetLeftMargin(left)
}

// line writes one wrapped, left-aligned paragraph in the given style, size,
// colour and line height. An empty style is regular; "B" is bold.
func (d *document) line(style string, size float64, c rgb, h float64, s string) {
	d.p.SetFont(fontFamily, style, size)
	d.p.SetTextColor(c.r, c.g, c.b)
	d.p.MultiCell(d.cw, h, s, "", "L", false)
}

// rgb is an 8-bit-per-channel colour for fpdf's int-triplet colour setters.
type rgb struct{ r, g, b int }

// joinNonEmpty joins only the non-empty parts with sep, so an absent optional
// field (an experience with no arrangement, say) leaves no dangling separator.
func joinNonEmpty(sep string, parts ...string) string {
	kept := make([]string, 0, len(parts))
	for _, p := range parts {
		if p != "" {
			kept = append(kept, p)
		}
	}
	return strings.Join(kept, sep)
}
