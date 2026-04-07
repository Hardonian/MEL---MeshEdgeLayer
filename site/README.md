# MEL public site (Next.js)

This folder contains a lightweight public-facing Next.js site for MEL orientation:

- Home (`/`)
- Quick start (`/quickstart`)
- Compare vs generic observability (`/compare`)
- Operator guide (`/guide`)
- Architecture primer (`/architecture`)
- Trust / privacy / security pointers (`/trust`)
- Help/orientation (`/help`)
- FAQ (`/faq`)
- Contribute (`/contribute`)
- Acknowledgements / dependencies (`/acknowledgements`)

Canonical documentation remains in the repository `docs/` tree; this site is a front door, not a second product manual.

The directory has its own `go.mod` so `go test ./...` at the repository root does not descend into `site/node_modules/` after `npm ci`.

It is **not** the operator console (that lives in `frontend/` and ships inside the `mel` binary).

## Run locally

```bash
. "$HOME/.nvm/nvm.sh" && nvm use 24
cd site
npm ci
npm run dev
```

## Build checks (same as CI)

```bash
npm run lint
npm run typecheck
npm run build
```

## Canonical URL for metadata (production)

Sitemap, `robots.txt`, and Open Graph `metadataBase` use **`SITE_CANONICAL_ORIGIN`** at build time. Default in code is the usual GitHub Pages base for this repo; override when you deploy elsewhere:

```bash
SITE_CANONICAL_ORIGIN=https://your.domain npm run build
```

From the repository root: `make site-verify` runs `npm ci`, lint, typecheck, and build for this folder (Node 24.x required).


## Vercel deploy contract (monorepo-safe)

This repository root does **not** contain a Node `package.json`, so Vercel must build the Next.js app in `site/` explicitly.

Use either:

1. **Project setting**: set Vercel **Root Directory** to `site`, or
2. Commit-level fallback (already included): root `vercel.json` with:
   - `installCommand`: `cd site && npm ci`
   - `buildCommand`: `cd site && npm run build`
   - `devCommand`: `cd site && npm run dev`

This prevents root-level 404 deployments caused by building the wrong directory.

