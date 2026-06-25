// Package site is the application layer that turns résumé content and a GitHub
// stats snapshot into the rendered static site.
package site

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/prorochestvo/whoami/internal/domain"
	"github.com/prorochestvo/whoami/internal/dto"
)

// StatsFetcher fetches a GitHub stats snapshot (implemented by infrastructure/github).
type StatsFetcher interface {
	Fetch(ctx context.Context) (domain.GitHubStats, error)
}

// ContentSource loads per-locale résumé content.
type ContentSource interface {
	Available() []string
	Load(locale string) (domain.Resume, error)
	Raw(locale string) ([]byte, error)
}

func NewBuilder(content ContentSource, fetcher StatsFetcher, renderer *Renderer, locales []string, siteURL string, logger *log.Logger) *Builder {
	return &Builder{
		content:  content,
		fetcher:  fetcher,
		renderer: renderer,
		locales:  locales,
		siteURL:  siteURL,
		logger:   logger,
		now:      func() time.Time { return time.Now().UTC() },
	}
}

type Builder struct {
	content  ContentSource
	fetcher  StatsFetcher
	renderer *Renderer
	locales  []string
	siteURL  string
	logger   *log.Logger
	now      func() time.Time
}

// Build fetches GitHub stats once (degrading gracefully), then renders a page and
// emits JSON for each locale. A single failing locale fails the whole build so the
// deployed site never has dangling hreflang links.
func (b *Builder) Build(ctx context.Context) error {
	// fetch stats once; the context budget is shared across all locales
	stats, err := b.fetcher.Fetch(ctx)
	if err != nil {
		b.logger.Printf("github stats unavailable, rendering without them: %v", err)
		stats = domain.GitHubStats{Available: false}
	}

	allMeta, rawByLocale, err := b.localeMeta()
	if err != nil {
		return err
	}

	renderedPages := make([]dto.Page, 0, len(b.locales))

	for _, locale := range b.locales {
		resume, err := b.content.Load(locale)
		if err != nil {
			return fmt.Errorf("build: load locale %s: %w", locale, err)
		}

		page := dto.NewPage(resume, stats, b.siteURL, locale, allMeta, b.now())

		if err := b.renderer.Render(page); err != nil {
			return fmt.Errorf("build: render locale %s: %w", locale, err)
		}

		if err := b.renderer.WriteLocaleJSON(locale, rawByLocale[locale]); err != nil {
			return fmt.Errorf("build: write json for locale %s: %w", locale, err)
		}

		renderedPages = append(renderedPages, page)
	}

	if err := b.renderer.CopyAssets(); err != nil {
		return fmt.Errorf("build: copy assets: %w", err)
	}

	if err := b.renderer.WriteSitemap(renderedPages, b.siteURL); err != nil {
		return fmt.Errorf("build: write sitemap: %w", err)
	}

	return nil
}

// localeMeta reads each locale's JSON once and returns the metadata plus a
// code→raw-bytes map the render loop reuses to avoid a second Raw() call.
func (b *Builder) localeMeta() ([]dto.LocaleMeta, map[string][]byte, error) {
	meta := make([]dto.LocaleMeta, 0, len(b.locales))
	rawByLocale := make(map[string][]byte, len(b.locales))
	for _, code := range b.locales {
		raw, err := b.content.Raw(code)
		if err != nil {
			return nil, nil, fmt.Errorf("build: localeName for %s: %w", code, err)
		}
		rawByLocale[code] = raw
		var header struct {
			LocaleName string `json:"localeName"`
		}
		if err := json.Unmarshal(raw, &header); err != nil {
			return nil, nil, fmt.Errorf("build: parse localeName for %s: %w", code, err)
		}
		name := header.LocaleName
		if name == "" {
			name = code
		}
		meta = append(meta, dto.LocaleMeta{Code: code, Name: name})
	}
	return meta, rawByLocale, nil
}
