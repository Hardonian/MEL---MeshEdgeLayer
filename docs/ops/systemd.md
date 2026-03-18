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
- `UMask=0077`

Important operator note:

- the unit is authored for `User=mel` and `Group=mel`,
- the service account must have access to the configured data directory,
- for direct serial deployments, the service account must also be in `dialout` or the equivalent serial-device group for the host distro.

Inspect logs with:

```bash
journalctl -u mel -f
```
