# Changelog

## Unreleased

### Added
- Operator-facing configuration, evaluation, and known-limitations docs that match current RC1 code paths.
- A stricter transport matrix covering serial direct-node, TCP direct-node, MQTT, unsupported transports, and control-path gaps.

### Changed
- README, architecture docs, install docs, security notes, and contributing guidance now explicitly distinguish implemented, partial, experimental, and unsupported surfaces.
- Config linting now warns when operators enable placeholder metrics settings, BLE experimental flags, or `storage.encryption_required` without actual MEL at-rest encryption.
- CLI/API empty-state outputs now prefer empty arrays over `null` where no observations exist yet.
- `features.web_ui=false` now disables the HTML UI route instead of being ignored.
- Doctor output now includes historical per-transport ingest observations without claiming that a transport is live when only prior SQLite evidence exists.
- Quickstart and smoke flows now tighten generated config permissions to avoid spurious doctor warnings during first-run validation.

### Fixed
- Restored the missing CLI doctor helper so `make build`, `go test ./...`, and operator-facing doctor summaries compile again.
