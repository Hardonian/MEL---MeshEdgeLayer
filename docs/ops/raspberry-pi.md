# Raspberry Pi notes

- Prefer a local serial-attached node as the primary deployment mode.
- Use stable serial paths under `/dev/serial/by-id/` when available.
- Keep MEL bound to `127.0.0.1` unless you have a deliberate remote-access design with auth.
- Run `mel doctor` after hardware changes, group membership changes, or node reattachment.
