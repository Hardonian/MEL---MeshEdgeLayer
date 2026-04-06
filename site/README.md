# MEL public site (Next.js)

This folder contains a lightweight public-facing Next.js site for MEL orientation:

- Home (`/`)
- Quick start (`/quickstart`)
- Help/orientation (`/help`)
- Contribute (`/contribute`)
- Acknowledgements/dependencies (`/acknowledgements`)

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
