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
