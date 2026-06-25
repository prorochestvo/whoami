// Command whoami generates the static portfolio site into ./build from embedded
// locale JSON and build-time GitHub stats (degrading gracefully on failure).
package main

import (
	"context"
	"log"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/prorochestvo/whoami/internal/application/site"
	"github.com/prorochestvo/whoami/internal/infrastructure/github"
	resumerepo "github.com/prorochestvo/whoami/internal/repository/resume"
)

func main() {
	logger := log.New(os.Stderr, "whoami: ", log.LstdFlags)

	githubUser := envOr("WHOAMI_GITHUB_USER", "prorochestvo")
	githubToken := os.Getenv("WHOAMI_GITHUB_TOKEN")
	siteURL := envOr("WHOAMI_SITE_URL", "http://localhost:8000/")

	// locales default to "en,ru", overridable via WHOAMI_LOCALES; the "qa"
	// placeholder is for local/CI testing only, never production.
	locales := parseLocales(envOr("WHOAMI_LOCALES", "en,ru"))

	available := resumerepo.Available()
	availSet := make(map[string]bool, len(available))
	for _, l := range available {
		availSet[l] = true
	}
	for _, l := range locales {
		if !availSet[l] {
			logger.Fatalf("locale %q is not available; embed a content/%s.json file first (available: %s)",
				l, l, strings.Join(available, ", "))
		}
	}

	if _, err := resumerepo.Load("en"); err != nil {
		logger.Fatalf("default locale 'en' failed to load: %v", err)
	}

	renderer, err := site.NewRenderer("templates", "web", "build")
	if err != nil {
		logger.Fatalf("init renderer: %v", err)
	}

	fetcher := github.NewClient(githubUser, githubToken)

	builder := site.NewBuilder(resumerepo.NewAdapter(), fetcher, renderer, locales, siteURL, logger)

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	if err := builder.Build(ctx); err != nil {
		logger.Fatalf("build site: %v", err)
	}
	logger.Printf("generated static site into ./build (locales: %s)", strings.Join(locales, ", "))
}

func parseLocales(raw string) []string {
	parts := strings.Split(raw, ",")
	locales := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			locales = append(locales, p)
		}
	}
	seen := make(map[string]bool)
	uniq := make([]string, 0, len(locales))
	for _, l := range locales {
		if !seen[l] {
			seen[l] = true
			uniq = append(uniq, l)
		}
	}
	// "en" first, then the rest alphabetically, so sitemap/alternate order is
	// deterministic regardless of input order
	sort.SliceStable(uniq, func(i, j int) bool {
		if uniq[i] == "en" {
			return true
		}
		if uniq[j] == "en" {
			return false
		}
		return uniq[i] < uniq[j]
	})
	return uniq
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
