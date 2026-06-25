# whoami

Personal portfolio / CV site for Seilbek Skindirov ([github.com/prorochestvo](https://github.com/prorochestvo)),
built as a static site by a Go generator and deployed to Cloudflare Pages.

> The repository is currently being scaffolded — some of the pieces described below
> are still landing. Anything not yet implemented is called out explicitly.

## How it works

There is no live server. `whoami` is a **build-time static-site generator**:

1. `cmd/whoami` reads the CV content, fetches GitHub stats over the API, and renders
   `html/template` sources into static HTML.
2. The HTML plus all static assets are written into `build/`.
3. `build/` is published to Cloudflare Pages.

GitHub stats are kept fresh by a **scheduled rebuild** (cron → Cloudflare deploy hook),
not by an always-on backend.

Why static instead of a live server:

- **Free & simple** — no runtime, no database, nothing to keep alive.
- **Fast** — everything is pre-rendered static files served from the edge.
- **Crawlable / good SEO** — content is server-rendered into HTML at build time, so
  crawlers see the real content, not a client-side render.
- **Fresh enough** — a scheduled rebuild refreshes GitHub stats (~daily) without
  paying for an always-on process.
- **Token stays in CI** — the GitHub token lives only as a Cloudflare Pages env secret
  and is never emitted into the generated output.

The GitHub client degrades gracefully: if there is no token or no network at build
time, it falls back rather than failing, so the build always produces output.

## Quick start

```bash
git clone https://github.com/prorochestvo/whoami.git
cd whoami

make build   # run the generator → build/
make run     # build, then serve build/ locally for preview
```

Then open the local preview URL printed by `make run` (e.g. http://localhost:8080).

### Make targets

| Target        | What it does                                           |
|---------------|--------------------------------------------------------|
| `make build`  | Runs the generator, writing output into `build/`.      |
| `make run`    | Builds, then serves `build/` locally for preview.      |
| `make test`   | `go fmt` + `go vet` + `go test ./...`.                 |
| `make format` | `go fmt ./...`.                                         |
| `make clean`  | Removes build artifacts.                                |

## Project structure

The layout is a DDD-style split (mirroring the `fx_rate_monitor` project):

| Path                                | Role                                                                       |
|-------------------------------------|---------------------------------------------------------------------------|
| `cmd/whoami/main.go`                | Generator entry point / composition root.                                 |
| `internal/domain/`                  | Pure CV value objects (no dependencies).                                   |
| `internal/repository/resume/`       | In-memory source of truth for the CV content (`Load()`).                   |
| `internal/infrastructure/github/`   | Build-time GitHub API client (degrades gracefully when offline / no token).|
| `internal/application/site/`        | `Builder` use-case + `Renderer` (templates → `build/`, copies `web/*`).    |
| `internal/dto/`                     | `Page` view model passed to templates.                                     |
| `templates/*.tmpl`                  | `html/template` sources (`layout` + `content`).                            |
| `web/`                              | Static source assets: `css/`, `js/`, `fonts/`, `img/`, `robots.txt`, `_headers`. |
| `build/`                            | Generated output (gitignored); Cloudflare Pages output directory.          |
| `plans/`                            | Markdown task plans (active → `completed/` → `history/`).                  |

- Module: `github.com/prorochestvo/whoami`
- Go: 1.26

## Configuration

| Variable               | Required | Purpose                                                                 |
|------------------------|----------|-------------------------------------------------------------------------|
| `WHOAMI_GITHUB_TOKEN`  | No       | Raises the GitHub API rate limit at build time. Injected as a Cloudflare Pages env secret in CI. |

- The GitHub user whose stats are fetched is `prorochestvo`.
- The token must **never** appear in generated output.
- Never commit a `.env` file.

## Deployment

Deployed via **Cloudflare Pages**:

- **Build command:** `make build`
- **Output directory:** `build/`
- **Environment secret:** `WHOAMI_GITHUB_TOKEN` set in the Pages project settings.
- **Stats refresh:** a scheduled job (cron) triggers a Cloudflare **deploy hook** to
  rebuild the site, pulling fresh GitHub stats — no always-on server required.

## Design

A code-editor / terminal aesthetic:

- **JetBrains Mono**, self-hosted in `web/fonts/` — no runtime CDN.
- Dark theme, near-monochrome with a single accent color.

## Front-end constraints

The generated front-end is intentionally minimal:

- Vanilla HTML / CSS / JS — no framework, no bundler, no npm dependencies, no WASM.
- Fonts are self-hosted; nothing is loaded from a runtime CDN.
- Content is server-rendered into HTML at build time, so the site is fast and crawlable.
- GitHub data injected into templates is auto-escaped by `html/template`.
