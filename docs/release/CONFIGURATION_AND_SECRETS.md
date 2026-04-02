# Configuration and Secrets

## Principles

- Configuration must fail closed when secret requirements are unmet.
- No silent fallback to insecure defaults for privileged features.
- Secrets should be provided via environment or protected files with strict permissions.

## Operator checklist

- Validate config before start: `mel config validate --config mel.json`.
- Restrict config file permissions.
- Keep auth/control settings explicit and reviewable.
- Record any secret-rotation actions in operations notes.

## Caveat

MEL currently validates certain encryption prerequisites but does not itself provide full at-rest DB encryption.
