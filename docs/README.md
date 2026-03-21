# MEL Documentation

Welcome to the MeshEdgeLayer (MEL) documentation hub. MEL is a local-first observability and control plane for Meshtastic networks.

## 🏁 Getting Started

If you are new to MEL, start here:

- [**Quickstart Index**](getting-started/README.md) - All setup and tour guides.
- [**Installation**](ops/install-linux.md) - Detailed setup steps for Linux (systemd focus).
- [**First 10 Minutes**](getting-started/first-10-minutes.md) - A guided tour of your new MEL instance.
- [**Hardware Support**](ops/support-matrix.md) - Verified nodes and transports.

## 🏗️ Architecture & Concepts

Understand how MEL works under the hood:

- [**System Overview**](architecture/overview.md) - High-level architecture and data flow.
- [**Subsystems**](architecture/layers.md) - Ingest, Intelligence, and Persistence layers.
- [**Trust Model**](architecture/CONTROL_PLANE_TRUST_MODEL.md) - How MEL handles security and remediation.
- [**Relay & Node Layout**](architecture/central-extension-node-layout.md) - Topology and deployment patterns.

## 🛠️ Operator Guide

Guides for running MEL in a production environment:

- [**Configuration Reference**](ops/configuration.md) - Detailed breakdown of `mel.json`.
- [**CLI Reference**](ops/cli-reference.md) - Mastering the `mel` command line.
- [**Diagnostics & Doctor**](ops/diagnostics.md) - Using `mel doctor` to maintain health.
- [**Incident Triage**](ops/incident-triage.md) - Handling mesh alerts and failures.
- [**Backup & Restore**](ops/runbooks.md) - Protecting your mesh data.
- [**Privacy Audit**](privacy/overview.md) - Managing your privacy posture.

## 🔌 Integration & API

Extend MEL or integrate it with other tools:

- [**JSON API Reference**](ops/api-reference.md) - Build your own dashboards and tools.
- [**Metrics**](ops/metrics.md) - Ingesting MEL data into Prometheus/Grafana.
- [**Protobuf Extensions**](contributor/protobuf-extension-guide.md) - Adding support for new Meshtastic payloads.

## 🤝 Project & Community

- [**Roadmap**](roadmap/ROADMAP_EXECUTION.md) - Current status and future plans.
- [**Contributing**](../CONTRIBUTING.md) - How to help improve MEL.
- [**Changelog**](../CHANGELOG.md) - Recent updates and fixes.
- [**Security Policy**](../SECURITY.md) - Reporting vulnerabilities.

---

*MEL — Truthful local-first mesh observability.*
