# MEL public site (Next.js)

This folder contains a lightweight public-facing Next.js site for MEL orientation:

- Home (`/`)
- Quick start (`/quickstart`)
- Architecture primer (`/architecture`)
- Trust / privacy / security pointers (`/trust`)
- Help/orientation (`/help`)
- FAQ (`/faq`)
- Contribute (`/contribute`)
- Acknowledgements / dependencies (`/acknowledgements`)

Canonical documentation remains in the repository `docs/` tree; this site is a front door, not a second product manual.

The directory has its own `go.mod` so `go test ./...` at the repository root does not descend into `site/node_modules/` after `npm ci`.

## Run locally

```bash
. "$HOME/.nvm/nvm.sh" && nvm use 24
cd site
npm install
npm run dev
```

## Build checks

```bash
npm run lint
npm run typecheck
npm run build
```
