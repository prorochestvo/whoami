// Package domain holds the CV value objects: pure data with no external
// dependencies, shared by the repository, application, and DTO layers.
package domain

type Person struct {
	Name       string
	Title      string
	Location   string
	Tagline    string
	BirthYear  int
	Experience string // human-readable total, e.g. "17+ years"
	GitHubUser string
}
