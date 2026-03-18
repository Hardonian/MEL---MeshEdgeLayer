# Contributing to MEL

## Ground rules

- Keep MEL honest: no fake transports, no mock mesh dashboards, no dead routes.
- Preserve stock Meshtastic compatibility.
- Prefer stdlib-only Go unless a dependency is already vendored.
- Make degraded states explicit in CLI, API, and UI.
- Add tests with any behavior change that affects operator truth, privacy, config, or persistence.

## Safe extension points

- policy/privacy logic
- exporter plugins
- alert plugins
- docs and operator workflows

Core transport ownership and storage semantics should be changed only when the implementation and tests stay truthful.

## Typical workflow

```bash
make verify
./scripts/smoke.sh
```

## Pull requests

Include:

- design intent
- operator impact
- verification evidence
- residual risk or limitations
