---
name: project-i18n-architecture
description: The decided architecture for JSON-driven i18n content + client-side language switcher (plan 001), and the non-obvious constraints that shaped it.
metadata:
  type: project
---

The whoami site is being moved from hardcoded Go résumé content to per-locale JSON as the
single source of truth, with SEO-safe per-locale pre-rendered pages and a
progressive-enhancement language switcher. Plan: `plans/001-json-i18n-content-and-language-switcher.md`.

**Why:** owner wants i18n-ready content and a client-side language switcher (fetch + CSS
progress bar) without sacrificing SEO. Architecture was decided by the user, not open for
re-litigation — plan the HOW.

**How to apply:** when extending or implementing this, honor these decided points:
- Default locale `en` renders to `build/index.html` (NO `/en/` prefix); others to
  `build/<lang>/index.html`. This preserves the already-indexed `/` URL — do not move
  `en` under `/en/`.
- Content is embedded via `//go:embed`. Embed path can't use `..`, so `content/` lives
  **under the owning package** (`internal/repository/resume/content/`), not module root.
- Domain structs stay pure (no `json` tags). A repository-local `resumeJSON` DTO +
  `toDomain` mapper owns the wire format. [[project-ddd-layer-rules]]
- The JSON emitted to `build/content/<lang>.json` for the runtime swap is the SAME source
  bytes the build render reads (copy raw bytes; never re-marshal `domain.Resume`, or the
  `locale`/`localeName` header fields drift).
- Phasing: Phase 1 (JSON source + emit + SEO-from-JSON, single locale, no visible change)
  ships before Phase 2 (locale loop, hreflang, switcher, progress bar).
