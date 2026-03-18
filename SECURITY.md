# Security Policy

## Supported posture

MEL RC1 is designed for local-first deployments. The supported baseline is:

- bind to localhost,
- expose UI/API remotely only when auth is enabled and the access path is reviewed,
- use one real transport at a time unless you have verified contention behavior,
- keep exports redacted by default,
- keep precise position storage disabled by default.

## Scope boundaries that matter for security review

- MEL does not currently provide radio admin/control operations.
- MEL does not provide at-rest SQLite encryption itself.
- MEL does not provide a metrics listener even though config placeholders exist.
- Restore is dry-run validation only.

## Reporting a vulnerability

Please open a private security report with:

- affected version or commit,
- exact config posture,
- reproduction steps,
- observed impact,
- any logs or backup validation output needed to understand the issue.

Do not post secrets, session material, or precise location data in public issues.

## Hardening notes

- MEL warns on broad config file permissions.
- MEL treats MQTT, map reporting, remote exposure, and long retention as privacy-sensitive posture.
- MEL does not claim security properties it cannot prove in code or tests.
