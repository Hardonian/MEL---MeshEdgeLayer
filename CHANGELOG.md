# Changelog

## Unreleased

### Added
- `mel init`, `mel node inspect`, `mel import validate`, `mel backup create`, and `mel backup restore --dry-run` commands.
- Versioned `/api/v1/*` endpoints for status, nodes, node detail, transport health, policy explain, and privacy audit.
- Plaintext-safe backup bundle creation with restore dry-run validation.
- Linux install, upgrade, uninstall, and Termux launcher scripts.
- Runtime, privacy, ops, product, and community documentation for MEL RC1.
- Configuration, transport matrix, first-10-minutes, and known-limitations docs aligned with current direct serial/TCP and MQTT reality.

### Changed
- Transport health now includes packet accounting, reconnect tracking, and last-error timestamps.
- Privacy audit and policy explanation outputs are structured for CLI, UI, and API reuse.
- Web UI now has truthful onboarding, transport health, privacy findings, recommendation, and event sections.
- Config validation and linting now highlight unsafe remote bind, risky MQTT posture, long retention, and multi-transport contention.
- Linux install guidance and install script now account for the `mel` service account and serial-group access expected by the shipped systemd unit.
