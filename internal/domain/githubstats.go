package domain

import "time"

// GitHubStats is a build-time snapshot of public GitHub activity. Available is
// false when it could not be fetched; consumers must still render the site.
type GitHubStats struct {
	Login        string
	PublicRepos  int
	Followers    int
	Following    int
	TotalStars   int
	TopLanguages []LanguageStat
	FetchedAt    time.Time
	Available    bool
}

type LanguageStat struct {
	Name  string
	Repos int
}
