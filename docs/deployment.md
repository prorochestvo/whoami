# Deployment

`whoami` is a build-time static site: a Go generator (`cmd/whoami`) renders the CV and
build-time GitHub stats into `build/`, which is published to **Cloudflare Pages**. There
is no server to run — no VPS, no systemd, no nginx.

## Primary path — GitHub Actions builds and deploys (Direct Upload)

GitHub Actions runs the generator and uploads `build/` to Cloudflare Pages with
`wrangler`. This is self-contained: it pins the Go version itself and does not rely on
Cloudflare's build image. Two workflows drive it:

- **`.github/workflows/ci.yml`** — on every pull request: `gofmt` check, `go vet`,
  `go test`, and a generator build. Keeps a red tree from reaching production.
- **`.github/workflows/deploy.yml`** — on push to `main`, on a monthly `schedule` (1st of
  the month, 00:00 UTC), and on manual `workflow_dispatch`: builds the site and runs
  `wrangler pages deploy build`, then ensures the custom domain is attached. The scheduled
  run refreshes the baked-in GitHub stats without a code change.

### One-time setup

1. **Create the Pages project** (Direct Upload, name must match `--project-name=whoami`):

   ```bash
   npx wrangler pages project create whoami --production-branch main
   ```

   Or in the dashboard: **Workers & Pages → Create → Pages → Direct Upload**, name it
   `whoami`.

2. **Get credentials:**
   - **Account ID** — Cloudflare dashboard → Workers & Pages → right sidebar.
   - **API token** — My Profile → API Tokens → Create Token → use the
     **"Cloudflare Pages — Edit"** template (scope it to your account).

3. **Add GitHub repository secrets** (Settings → Secrets and variables → Actions):

   | Name | Type | Value |
   |------|------|-------|
   | `CLOUDFLARE_API_TOKEN` | secret | the Pages:Edit API token |
   | `CLOUDFLARE_ACCOUNT_ID` | secret | your account ID |
   | `WHOAMI_SITE_URL` | **variable** | production URL, e.g. `https://whoami.pages.dev/` (trailing slash) |

   The GitHub stats fetch uses the workflow's built-in `GITHUB_TOKEN` automatically — no
   extra secret is needed, and a failed fetch degrades gracefully.

4. **Push to `main`** (or run the *deploy* workflow manually). The site goes live at
   `https://whoami.pages.dev`.

### Custom domain

Cloudflare Pages → the `whoami` project → **Custom domains** → add your domain and follow
the DNS prompt. Then update the `WHOAMI_SITE_URL` repository variable to match so
canonical and OpenGraph tags point at the real URL, and redeploy.

## Alternative — Cloudflare Pages Git integration

If you prefer Cloudflare to build on its own (this gives automatic **per-PR preview
deployments**), connect the repo instead of using `wrangler`:

- **Build command:** `make build`
- **Build output directory:** `build/`
- **Environment variables:** `WHOAMI_GITHUB_TOKEN` (secret, optional),
  `WHOAMI_SITE_URL`, and `GO_VERSION=1.26` (pin it — Cloudflare's default Go may be older).

Then keep stats fresh with a **Deploy Hook** (project → Settings → Builds & deployments →
Deploy hooks) triggered on a schedule — either a Cloudflare Worker Cron Trigger or a
GitHub Actions `schedule:` job that `curl -X POST`s the hook URL. Store the hook URL as a
secret; never commit it.

Trade-off: this path depends on the Go toolchain available in Cloudflare's build image.
If `GO_VERSION=1.26` is not yet supported there, use the primary `wrangler` path above.

## Secrets & variables summary

| Where | Name | Purpose |
|-------|------|---------|
| GitHub secret | `CLOUDFLARE_API_TOKEN` | wrangler auth (Pages:Edit) |
| GitHub secret | `CLOUDFLARE_ACCOUNT_ID` | target account |
| GitHub variable | `WHOAMI_SITE_URL` | canonical / OpenGraph base URL |
| (auto) | `GITHUB_TOKEN` | raises GitHub API rate limit during the build |

The GitHub token (any form) must never appear in generated output or be committed.
