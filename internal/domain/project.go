package domain

// Project is one portfolio entry. An empty URL renders without a link (NDA or
// private repo); Repo is the label shown opposite the name (repo path or a note).
type Project struct {
	Name       string
	Repo       string
	URL        string
	Stack      []string
	Summary    string
	Highlights []string
}
