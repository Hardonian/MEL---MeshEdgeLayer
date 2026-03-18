# Security Policy

## Supported posture

MEL RC1 is designed for local-first deployments. The supported default is:

- bind to localhost
- use one real transport at a time unless you have verified contention behavior
- keep auth enabled for any remote exposure
- keep exports redacted by default
- keep precise position storage disabled by default

## Reporting a vulnerability

Please open a private security report with:

- affected version or commit
- exact config posture
- reproduction steps
- observed impact
- any logs or backup validation output needed to understand the issue

Do not post secrets, session material, or precise location data in public issues.

## Hardening notes

- MEL warns on broad config file permissions.
- MEL treats MQTT, map reporting, and long retention as privacy-sensitive.
- MEL does not claim security properties it cannot prove in code or tests.
