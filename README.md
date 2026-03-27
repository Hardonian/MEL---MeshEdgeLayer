# MEL — MeshEdgeLayer

**Truthful, local-first mesh observability and operator control plane.**

![MEL Hero](assets/mel_hero_new_1774058412698.png)

[![Go Report Card](https://goreportcard.com/badge/github.com/mel-project/mel)](https://goreportcard.com/report/github.com/mel-project/mel)
[![License](https://img.shields.io/github/license/mel-project/mel)](LICENSE)
[![Status](https://img.shields.io/badge/status-0.1.0--rc1-blue.svg)](docs/roadmap/ROADMAP_EXECUTION.md)

---

## What is MEL?

MEL is a heavy-duty ingest, persistence, and observability layer designed for **production-oriented Meshtastic deployments**. It provides operators with high-fidelity visibility into mesh health, packet traffic, and node telemetry without relying on cloud services or external dependencies.

Unlike generic dashboards, MEL is built on a **"Truth First" Philosophy**: it only reports data it has successfully persisted and verified in its local state. If MEL says it happened, it happened on the wire.

### Core Value Proposition

- **The "Black Box" Problem**: Generic mesh dashboards often hide transport failures or invent "healthy" traffic. MEL makes every degraded state explicit.
- **Operator Ownership**: Your mesh data belongs in your SQLite database, not a third-party cloud.
- **Relentless Persistence**: Every packet is checked, classified, and stored with an audit trail.
- **Guarded Automation**: MEL doesn't just watch; its [Control Plane](docs/architecture/control-plane.md) suggests and executes safe remediation to keep your mesh alive.

---

## Key Capabilities

- **Multi-Transport Ingest**: Simultaneous support for **Serial (USB)**, **TCP (Network)**, and **MQTT** transports.
- **Authoritative Diagnostics**: Run `mel doctor` to verify host permissions, database integrity, and transport health in seconds.
- **Modern Operator UI**: A sleek, real-time Web Dashboard and a responsive TUI for field operations.
- **Intelligence Layer**: Deep packet inspection that classifies traffic into `text`, `position`, `node_info`, and `telemetry`.
- **Privacy by Design**: Built-in redaction, privacy audits, and local-only position storage by default.

---

## ⚡ Quickstart (Under 5 Minutes)

MEL is designed to be operational before your next packet arrives.

### 1. Build & Install

```bash
go build -o mel ./cmd/mel
```

### 2. Initialize & Validate

```bash
# Generate a fresh operator config
./mel init --config configs/mel.json

# Run environment diagnostics (Config, DB, Transports)
./mel doctor --config configs/mel.json
```

### 3. Launch the Control Plane

```bash
# Start the ingest engine and web dashboard
./mel serve --config configs/mel.json
```
Visit **[http://localhost:8080](http://localhost:8080)** to see your mesh come alive.

---

## What MEL is NOT (Honest Boundaries)

- **Not a Mesh Stack**: MEL does not implement Meshtastic routing or transmit radio packets itself.
- **No Mystery Interpolation**: 0 messages means 0 messages. We do not "guess" mesh state.
- **No Cloud Required**: MEL is local-first and works entirely offline.
- **No Hidden Randomness**: Decisions are grounded in deterministic, inspectable evidence.

Review our full [Known Limitations](docs/ops/limitations.md) for a detailed support matrix.

---

## 🏗️ Architecture

MEL follows a unidirectional, guarded data flow to ensure integrity.

```mermaid
graph TD
    subgraph "Meshtastic Mesh"
        N1[Node 1] --- N2[Node 2]
        N2 --- N3[Node 3]
    end

    subgraph "MEL Transports"
        SR[Serial / USB]
        TC[TCP / Network]
        MQ[MQTT Feed]
    end

    subgraph "MEL Core Engine"
        IG[Ingest Worker]
        DB[(SQLite Persistence)]
        IL[[Intelligence Layer]]
    end

    subgraph "Operator Interface"
        CLI[mel doctor / status]
        TUI[Terminal UI]
        WEB[Web Dashboard]
        API[JSON API / Metrics]
    end

    N1 --> SR
    N2 --> TC
    N3 --> MQ

    SR --> IG
    TC --> IG
    MQ --> IG

    IG --> DB
    DB <--> IL
    IL --> API
    DB --> CLI
    DB --> TUI
    DB --> WEB
```

---

## 📖 Next Steps in the Documentation

- [**Installation Guide**](docs/getting-started/installation.md)
- [**Full CLI Reference**](docs/ops/cli-reference.md)
- [**API Reference**](docs/ops/api-reference.md)
- [**Operator Runbooks**](docs/runbooks/README.md)
- [**Troubleshooting**](docs/ops/troubleshooting.md)

---

## Contributing

We welcome contributions that increase structural coherence and reduce entropy.

- **Bug Reports**: Open a [Bug Report](https://github.com/mel-project/mel/issues/new?template=bug_report.md).
- **Guidelines**: Read our [CONTRIBUTING.md](CONTRIBUTING.md).

MEL is licensed under the **Apache-2.0 License**.
© 2026 Hardonian / MeshEdgeLayer Contributors.
