# MEL public site (Next.js)

This folder contains a lightweight public-facing Next.js site for MEL orientation:

- Home (`/`)
- Quick start (`/quickstart`)
- Help/orientation (`/help`)
- Contribute (`/contribute`)
- Acknowledgements/dependencies (`/acknowledgements`)

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

From the repo root: `make site-verify` (clean install + lint + typecheck + build).

## Deploy / absolute URLs

Set **`NEXT_PUBLIC_SITE_URL`** to the deployed origin at build time (no trailing slash required), for example `https://your-host.example`. If unset, metadata, `robots.txt`, and `sitemap.xml` default to `http://localhost:3000` — appropriate for local preview only, not a fake production domain.
