# MEL Frontend

The MEL frontend is an operator workbench for incident intelligence and trusted control.

## Identity contract

The frontend is intentionally:

- dark by default;
- signal-first and high-density;
- explicit about degraded/partial/unknown state;
- restrained in motion and visual effects;
- grounded in operator truth (no claim inflation).

Canonical guidance:

- `docs/product/operator-vibe-system.md`
- `docs/contributor/writing-style-guide.md`

## Stack

1. React + Vite + TypeScript
2. Tailwind CSS + CSS variable semantic tokens (`src/index.css`, `tailwind.config.js`)
3. Lucide iconography
4. API/data hooks in `src/hooks/useApi.tsx`

## Design-system implementation notes

- Semantic color tokens are defined in `src/index.css` and mapped in `tailwind.config.js`.
- Status semantics (`live`, `degraded`, `critical`, `unsupported`, etc.) must remain stable across surfaces.
- Monospace usage is reserved for IDs, telemetry, timestamps, and command-like controls.
- `prefers-reduced-motion` is respected globally.

## Local development

Backend must run on `:8080`.

```bash
./bin/mel serve --config configs/mel.generated.json
```

Frontend (Node 24.x required):

```bash
. "$HOME/.nvm/nvm.sh" && nvm use 24
cd frontend
npm ci
npm run dev
```

## Verification

Common commands:

- `make lint`
- `make test`
- `make build`
- `make frontend-test-fast`

Treat failures as evidence: narrow claims or fix implementation.
