# MEL (MeshEdgeLayer)

**Evidence-first operator OS for mesh incidents and trusted control.**

## What It Is
An operator console for mesh network incidents that tracks explicit degraded states — not "live" by default. MEL says "unknown" when it can't prove otherwise.

## Who It's For
- Emergency response teams
- Field operations (off-grid, rural)
- Mesh network operators
- Critical infrastructure

## Core Features
- Explicit idle/degraded/unknown status
- Audit-grade control actions
- Incident handoff + ownership
- Topology intelligence
- Break-glass with SoD

## Quick Start
```bash
git clone https://github.com/Hardonian/MEL-MeshEdgeLayer
cd MEL-MeshEdgeLayer
make build
./bin/mel init --config config.json
./bin/mel serve
```

## Architecture
- Go binary for operators
- SQLite for local persistence  
- Optional PostgreSQL for fleet
- Web UI at `/`

## Pricing
- Open Source: GPL-3.0
- Enterprise: Contact for support

## Links
- Docs: `/docs`
- GitHub: https://github.com/Hardonian/MEL-MeshEdgeLayer
