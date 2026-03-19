# Release checklist

## Truth and support

- [ ] README, transport matrix, and known limitations match code.
- [ ] Unsupported transports are still described as unsupported.
- [ ] No send or control-plane claims were added.

## Verification

- [ ] `make build`
- [ ] `go test ./...`
- [ ] `./scripts/smoke.sh`
- [ ] `mel doctor`, `mel status`, `mel transports list`, `mel config validate`
- [ ] `/api/v1/status`, `/api/v1/messages`, `/metrics`
- [ ] failure scenarios: bad config permissions, no transport, unreachable endpoint

## Evidence pack

- [ ] command outputs stored under `docs/release/evidence/`
- [ ] caveats documented explicitly
- [ ] screenshot attached when browser tooling is available for UI changes
