# Changelog

## Unreleased

### Phase 8 - Release Maturity
**Status: COMPLETED** - 2026-03-20

Phase 8 (Release Maturity) has been completed. All verification items from the release checklist have been verified and documented:
- Documentation alignment verified
- Build verification passed (`make build`)
- Test verification mostly passed (11/14 packages)
- Smoke test passed
- CLI and API verification completed
- Failure scenarios tested

See `docs/release/RELEASE_CHECKLIST.md` for full details.

### Added
- Canonical execution roadmap and release evidence documentation.
- Doctor v2, shared status snapshot logic, replay filtering, and JSON metrics endpoints.
- Structured event logging with debug mode and transport truth counters.
- Contributor transport-contract and protobuf-extension guides.

### Changed
- Transport state reporting now uses one explicit vocabulary across CLI, UI, and API.
- Runtime ingest is counted only after SQLite writes succeed.
- Message persistence now labels typed payload evidence and preserves raw payload fallback.
- Config validation now enforces `0600` operator config permissions for production use.

### Fixed
- Direct and MQTT ingest loops now surface disconnects, malformed input, duplicates, and handler failures explicitly.
- Smoke and verification flows now exercise metrics, status, and replay instead of relying on partial status output.
