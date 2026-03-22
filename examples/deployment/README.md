# Deployment examples (production-honest)

These files document **real** MEL behaviors shipped in this repository. They are not a managed distribution.

## Config precedence (effective settings)

Order applied by `config.Load`:

1. JSON config file
2. Named profile overlay (`--profile` or `MEL_CONFIG_PROFILE`) from `configs/profiles/<name>.json`
3. Environment variables (`MEL_*` — see `internal/config/config.go` `applyEnv`)
4. Defaults (`config.Default()`)

CLI flags only select which config file path to load; they do not override individual JSON keys except via env after load.

## Commands

- `mel init` — write a starter config (0600).
- `mel bootstrap run|validate` — create data dir, apply migrations, optional dry-run validate.
- `mel doctor` — config + DB + schema compatibility + audit chain spot-check + transports.
- `mel upgrade preflight` — structured readiness report (also `GET /api/v1/health/upgrade`).
- `mel audit verify` — verify `audit_logs` tamper-evident chain (also `GET /api/v1/audit/verify`).

## Auth with API keys

When `auth.enabled` is true, set comma-separated keys in `MEL_AUTH_API_KEYS` or point `auth.api_keys_env` at an env var name whose value is comma-separated keys. Clients send `X-API-Key: <secret>`. If keys are configured, anonymous access is denied even when UI Basic credentials exist.

## Files

| File | Purpose |
|------|---------|
| `mel.systemd.service` | Example unit; adjust `User`, paths, and config. |
| `mel.env.example` | Environment overrides for production-style installs. |
| `docker-compose.yml` | Optional container layout; requires local image build. |
| `Dockerfile` | Minimal build from repo root. |
| `reverse-proxy.md` | Notes for TLS termination in front of MEL. |
