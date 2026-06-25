// Package github fetches public GitHub statistics for a user at build time and
// maps them into a domain.GitHubStats snapshot. Every failure degrades to an
// unavailable snapshot plus an error so the site build never fails because of it.
package github

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"time"

	"github.com/prorochestvo/whoami/internal/domain"
)

const DefaultAPIBaseURL = "https://api.github.com"

// NewClient returns a stats client for user. token is optional (it only raises
// the rate limit) and is used solely on the Authorization header — never stored,
// logged, or written into generated output.
func NewClient(user, token string) *Client {
	return &Client{
		user:    user,
		token:   token,
		baseURL: DefaultAPIBaseURL,
		http:    &http.Client{Timeout: 10 * time.Second},
		topN:    6,
	}
}

type Client struct {
	user    string
	token   string
	baseURL string
	http    *http.Client
	topN    int
}

// Fetch returns a stats snapshot for the configured user. On any failure it
// returns Available=false plus a non-nil error so the caller can log and continue.
func (c *Client) Fetch(ctx context.Context) (domain.GitHubStats, error) {
	if c.user == "" {
		return domain.GitHubStats{}, fmt.Errorf("github: empty user")
	}

	var profile struct {
		Login       string `json:"login"`
		PublicRepos int    `json:"public_repos"`
		Followers   int    `json:"followers"`
		Following   int    `json:"following"`
	}
	if err := c.getJSON(ctx, "/users/"+c.user, &profile); err != nil {
		return domain.GitHubStats{}, fmt.Errorf("github: fetch profile: %w", err)
	}

	var repos []struct {
		Stargazers int    `json:"stargazers_count"`
		Language   string `json:"language"`
		Fork       bool   `json:"fork"`
	}
	if err := c.getJSON(ctx, "/users/"+c.user+"/repos?per_page=100&type=owner&sort=pushed", &repos); err != nil {
		return domain.GitHubStats{}, fmt.Errorf("github: fetch repos: %w", err)
	}

	stars := 0
	byLanguage := make(map[string]int)
	for _, r := range repos {
		if r.Fork {
			continue
		}
		stars += r.Stargazers
		if r.Language != "" {
			byLanguage[r.Language]++
		}
	}

	return domain.GitHubStats{
		Login:        profile.Login,
		PublicRepos:  profile.PublicRepos,
		Followers:    profile.Followers,
		Following:    profile.Following,
		TotalStars:   stars,
		TopLanguages: topLanguages(byLanguage, c.topN),
		FetchedAt:    time.Now().UTC(),
		Available:    true,
	}, nil
}

func (c *Client) getJSON(ctx context.Context, path string, out any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+path, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "whoami-generator")
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status %d", resp.StatusCode)
	}
	return json.NewDecoder(resp.Body).Decode(out)
}

// topLanguages returns the counts sorted by count descending, capped at n. Ties
// break alphabetically so the output is stable across builds.
func topLanguages(counts map[string]int, n int) []domain.LanguageStat {
	stats := make([]domain.LanguageStat, 0, len(counts))
	for name, repos := range counts {
		stats = append(stats, domain.LanguageStat{Name: name, Repos: repos})
	}
	sort.Slice(stats, func(i, j int) bool {
		if stats[i].Repos != stats[j].Repos {
			return stats[i].Repos > stats[j].Repos
		}
		return stats[i].Name < stats[j].Name
	})
	if n > 0 && len(stats) > n {
		stats = stats[:n]
	}
	return stats
}
