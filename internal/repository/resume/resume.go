// Package resume loads CV content from embedded content/<locale>.json files and
// validates it into domain.Resume. The embed path is local to this package, so
// content/ must not be moved out of this subtree.
package resume

import (
	"embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"sort"
	"strings"

	"github.com/prorochestvo/whoami/internal/domain"
)

//go:embed content/*.json
var contentFS embed.FS

func NewAdapter() *Adapter { return &Adapter{} }

// Adapter exposes the package-level resume functions as a site.ContentSource,
// so the site package needn't import this repository.
type Adapter struct{}

func (a *Adapter) Available() []string                       { return Available() }
func (a *Adapter) Load(locale string) (domain.Resume, error) { return Load(locale) }
func (a *Adapter) Raw(locale string) ([]byte, error)         { return Raw(locale) }

func Load(locale string) (domain.Resume, error) {
	raw, err := fs.ReadFile(contentFS, "content/"+locale+".json")
	if err != nil {
		return domain.Resume{}, fmt.Errorf("resume: load %s: %w", locale, err)
	}

	var dto resumeJSON
	if err := json.Unmarshal(raw, &dto); err != nil {
		return domain.Resume{}, fmt.Errorf("resume: parse %s: %w", locale, err)
	}

	if err := validate(dto); err != nil {
		return domain.Resume{}, fmt.Errorf("resume: validate %s: %w", locale, err)
	}

	return toDomain(dto), nil
}

// Available returns the locale codes that have a content file, sorted ascending
// for deterministic build output.
func Available() []string {
	entries, err := fs.ReadDir(contentFS, "content")
	if err != nil {
		return nil
	}
	locales := make([]string, 0, len(entries))
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if strings.HasSuffix(name, ".json") {
			locales = append(locales, strings.TrimSuffix(name, ".json"))
		}
	}
	sort.Strings(locales)
	return locales
}

// Raw returns the embedded JSON bytes for the locale unparsed, so the build and
// the client-side fetch share one byte source.
func Raw(locale string) ([]byte, error) {
	b, err := fs.ReadFile(contentFS, "content/"+locale+".json")
	if err != nil {
		return nil, fmt.Errorf("resume: raw %s: %w", locale, err)
	}
	return b, nil
}

// resumeJSON mirrors the on-disk JSON schema, keeping json tags out of domain.
type resumeJSON struct {
	Locale     string           `json:"locale"`
	LocaleName string           `json:"localeName"`
	Person     personJSON       `json:"person"`
	About      []string         `json:"about"`
	Projects   []projectJSON    `json:"projects"`
	Experience []experienceJSON `json:"experience"`
	Education  []educationJSON  `json:"education"`
	Skills     []skillGroupJSON `json:"skills"`
	Languages  []languageJSON   `json:"languages"`
	Relocation []string         `json:"relocation"`
	Contacts   []contactJSON    `json:"contacts"`
}

type personJSON struct {
	Name       string `json:"name"`
	Title      string `json:"title"`
	Location   string `json:"location"`
	Tagline    string `json:"tagline"`
	BirthYear  int    `json:"birthYear"`
	Experience string `json:"experience"`
	GitHubUser string `json:"githubUser"`
}

type experienceJSON struct {
	Company     string   `json:"company"`
	Role        string   `json:"role"`
	Period      string   `json:"period"`
	Location    string   `json:"location"`
	Arrangement string   `json:"arrangement"`
	Summary     string   `json:"summary"`
	Highlights  []string `json:"highlights"`
	Stack       []string `json:"stack"`
	Current     bool     `json:"current"`
	Start       string   `json:"start"`
	End         string   `json:"end"`
}

type projectJSON struct {
	Name       string   `json:"name"`
	Repo       string   `json:"repo"`
	URL        string   `json:"url"`
	Stack      []string `json:"stack"`
	Summary    string   `json:"summary"`
	Highlights []string `json:"highlights"`
}

type educationJSON struct {
	Year        string `json:"year"`
	Institution string `json:"institution"`
	Field       string `json:"field"`
	Degree      string `json:"degree"`
}

type skillGroupJSON struct {
	Name  string   `json:"name"`
	Items []string `json:"items"`
}

type languageJSON struct {
	Name  string `json:"name"`
	Level string `json:"level"`
}

type contactJSON struct {
	Label string `json:"label"`
	Value string `json:"value"`
	URL   string `json:"url"`
}

func validate(r resumeJSON) error {
	if r.Person.Name == "" {
		return fmt.Errorf("person.name is empty")
	}
	if r.Person.GitHubUser == "" {
		return fmt.Errorf("person.githubUser is empty")
	}
	if len(r.Experience) == 0 {
		return fmt.Errorf("experience list is empty")
	}
	current := 0
	for _, e := range r.Experience {
		if e.Current {
			current++
		}
	}
	if current != 1 {
		return fmt.Errorf("expected exactly one current:true experience entry, found %d", current)
	}
	return nil
}

// lessExperience reports whether a should sort before b: most recent first by
// Start (ISO YYYY-MM) descending; ties broken by End descending where an empty
// End means "present" and ranks highest; remaining ties broken by Company name
// ascending. An empty Start ranks last so unkeyed entries sink to the bottom.
func lessExperience(a, b domain.Experience) bool {
	if a.Start != b.Start {
		if a.Start == "" {
			return false
		}
		if b.Start == "" {
			return true
		}
		return a.Start > b.Start
	}
	if a.End != b.End {
		// empty End means present — it ranks above any finished role
		if a.End == "" {
			return true
		}
		if b.End == "" {
			return false
		}
		return a.End > b.End
	}
	return a.Company < b.Company
}

// toDomain maps the DTO to the domain aggregate, sorting experience by
// lessExperience so order is driven by the data, not JSON file order.
func toDomain(r resumeJSON) domain.Resume {
	exp := make([]domain.Experience, len(r.Experience))
	for i, e := range r.Experience {
		exp[i] = domain.Experience{
			Company:     e.Company,
			Role:        e.Role,
			Period:      e.Period,
			Location:    e.Location,
			Arrangement: e.Arrangement,
			Summary:     e.Summary,
			Highlights:  e.Highlights,
			Stack:       e.Stack,
			Current:     e.Current,
			Start:       e.Start,
			End:         e.End,
		}
	}
	sort.SliceStable(exp, func(i, j int) bool {
		return lessExperience(exp[i], exp[j])
	})

	projects := make([]domain.Project, len(r.Projects))
	for i, p := range r.Projects {
		projects[i] = domain.Project{
			Name:       p.Name,
			Repo:       p.Repo,
			URL:        p.URL,
			Stack:      p.Stack,
			Summary:    p.Summary,
			Highlights: p.Highlights,
		}
	}

	edu := make([]domain.Education, len(r.Education))
	for i, e := range r.Education {
		edu[i] = domain.Education{
			Year:        e.Year,
			Institution: e.Institution,
			Field:       e.Field,
			Degree:      e.Degree,
		}
	}

	skills := make([]domain.SkillGroup, len(r.Skills))
	for i, s := range r.Skills {
		skills[i] = domain.SkillGroup{Name: s.Name, Items: s.Items}
	}

	langs := make([]domain.Language, len(r.Languages))
	for i, l := range r.Languages {
		langs[i] = domain.Language{Name: l.Name, Level: l.Level}
	}

	contacts := make([]domain.Contact, len(r.Contacts))
	for i, c := range r.Contacts {
		contacts[i] = domain.Contact{Label: c.Label, Value: c.Value, URL: c.URL}
	}

	return domain.Resume{
		Person: domain.Person{
			Name:       r.Person.Name,
			Title:      r.Person.Title,
			Location:   r.Person.Location,
			Tagline:    r.Person.Tagline,
			BirthYear:  r.Person.BirthYear,
			Experience: r.Person.Experience,
			GitHubUser: r.Person.GitHubUser,
		},
		About:      r.About,
		Projects:   projects,
		Experience: exp,
		Education:  edu,
		Skills:     skills,
		Languages:  langs,
		Relocation: r.Relocation,
		Contacts:   contacts,
	}
}
