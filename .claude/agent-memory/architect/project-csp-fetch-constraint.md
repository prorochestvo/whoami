---
name: project-csp-fetch-constraint
description: web/_headers ships a strict CSP; any client-side fetch/XHR needs connect-src 'self' added explicitly, and no inline script/style is allowed.
metadata:
  type: project
---

`web/_headers` ships a strict Content-Security-Policy: `default-src 'self'`,
`script-src 'self'`, `style-src 'self'`, `font-src 'self'`, `img-src 'self' data:`,
`base-uri 'none'`, `frame-ancestors 'none'`. There is NO explicit `connect-src`.

**Why:** the site is locked down by design (no framework/bundler, self-hosted everything).
`default-src 'self'` currently covers same-origin `fetch`, but relying on the fallback is
fragile — a future CSP tweak could silently kill any client-side fetch (e.g. the language
switcher fetching `/content/<lang>.json`).

**How to apply:** any feature that adds a client-side `fetch`/XHR must add an explicit
`connect-src 'self'` to `web/_headers`. No inline `<script>` or inline `<style>` / inline
style attributes are permitted (no `'unsafe-inline'`) — all JS/CSS must be in external
files under `web/`, and no `eval`/`new Function`. When editing the CSP, change only the
one directive you need; never loosen `script-src`/`style-src`/`default-src`.
