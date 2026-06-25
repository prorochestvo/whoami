---
name: project-i18n-architecture
description: Key decisions from plan 001 — JSON i18n content and language switcher — locked and implemented
metadata:
  type: project
---

Content lives at `internal/repository/resume/content/<locale>.json` embedded via `//go:embed`.
Default locale `en` renders to `build/index.html` (no `/en/` prefix). Other locales to `build/<lang>/index.html`.
JSON raw bytes copied verbatim to `build/content/<lang>.json` — never re-marshalled.
`ContentSource` interface defined in `site` package (consumer-defined): `Available() []string; Load(string) (domain.Resume, error); Raw(string) ([]byte, error)`.
Repository exposes `NewAdapter()` returning `*Adapter` that satisfies the interface without importing `site`.
GitHub stats fetched once before locale loop; a failed fetch renders all locales gracefully.
Any broken locale (including non-default) fails the whole build.
Locale set wired in `cmd/whoami/main.go` via `WHOAMI_LOCALES` env var (default `"en"`).
Placeholder locale `qa` exists for testing the multi-locale pipeline; never ships in production.
Sitemap generated at `build/sitemap.xml` from rendered pages. `robots.txt` has hardcoded `Sitemap:` line pointing at `https://prorochestvo.pages.dev/sitemap.xml`.
JS switcher uses `textContent`/`createElement` only — never `innerHTML` on fetched data.
Progress bar driven by `body.lang-loading` / `body.lang-done` classes; `prefers-reduced-motion` suppresses animation in CSS.

**Why:** plan 001 was fully implemented 2026-06-20; all 11 tasks done.
**How to apply:** When extending locale support (adding real translations), drop a new `<lang>.json` in `internal/repository/resume/content/` and add it to `WHOAMI_LOCALES`.
