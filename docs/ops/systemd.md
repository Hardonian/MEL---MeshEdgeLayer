# systemd operation

The shipped unit lives at `docs/ops/systemd/mel.service`.

## What it assumes

- `mel` binary at `/usr/local/bin/mel`
- config at `/etc/mel/mel.json`
- data dir at `/var/lib/mel`
- a `mel` service user/group already exist

## Hardening already present

- `NoNewPrivileges=true`
- `ProtectSystem=strict`
- `ProtectHome=true`
- `PrivateTmp=true`
- `ProtectKernelTunables=true`
- `ProtectControlGroups=true`
- `ProtectKernelModules=true`
- `LockPersonality=true`
- `MemoryDenyWriteExecute=true`
- `UMask=0077`

## What operators still must do

- create the service user/group,
- grant serial access if using direct-node serial mode,
- ensure `/var/lib/mel` is writable by that account,
- review whether `/etc/mel` should remain writable at runtime for your deployment.

Inspect logs with:

```bash
journalctl -u mel -f
```
