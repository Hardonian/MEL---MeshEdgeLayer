# Frontend contribution

## Preconditions

- **Node.js 24.x** only (`frontend/.nvmrc`, `frontend/package.json` engines). From repo root: `. ./scripts/dev-env.sh`.
- Backend on `http://127.0.0.1:8080` when exercising live API behavior (`./bin/mel serve ...`).

## Commands

```bash
cd frontend
npm ci          # or rely on make frontend-install from repo root
npm run dev     # Vite dev server (default http://localhost:5173)
npm run lint
npm run typecheck
npm run test
npm run build
```

Production embedding: `make build` copies `frontend/dist/*` into `internal/web/assets/`.

## Code layout

| Concern | Location |
| --- | --- |
| Routes / pages | `src/pages/` |
| Design-system-style primitives | `src/components/ui/` |
| API polling | `src/hooks/useApi.tsx`, `src/hooks/useIncidents.ts`, etc. |
| Operator truth copy helpers | `src/utils/evidenceSemantics.ts`, `src/utils/operatorWorkflow.ts` |

## UX doctrine (non-negotiable)

- **No fake live state**: empty and degraded states are valid product states.
- **Topology and maps**: copy must reflect ingest-derived context, not guaranteed RF paths (align with [terminology](../repo-os/terminology.md)).
- **Control surfaces**: submission ≠ approval ≠ execution; UI must not imply otherwise.

## Verification before PR

```bash
make frontend-typecheck
make frontend-test
make lint
make build
```

For release-grade confidence: `make premerge-verify`.
