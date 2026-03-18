# systemd operation

The shipped unit lives at `docs/ops/systemd/mel.service` and enables:

- `NoNewPrivileges=true`
- `ProtectSystem=strict`
- `ProtectHome=true`
- `PrivateTmp=true`
- `ProtectKernelTunables=true`
- `ProtectControlGroups=true`
- `LockPersonality=true`
- `MemoryDenyWriteExecute=true`

Inspect logs with:

```bash
journalctl -u mel -f
```
