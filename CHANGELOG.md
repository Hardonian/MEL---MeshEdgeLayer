# Changelog

## Unreleased

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
