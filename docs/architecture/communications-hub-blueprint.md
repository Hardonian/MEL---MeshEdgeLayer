# MEL communications hub blueprint (self-hosted, privacy-first)

## Status
Accepted foundation slice (March 29, 2026).

## Product thesis
MEL runs as a local-first operations hub for mixed local networks (mesh, LoRa, Wi‑Fi, Bluetooth context) with evidence-first incident workflows. MEL-owned logic stays focused on truth, evidence, incident intelligence, proofpacks, action memory, and operator auditability.

## Build vs borrow boundaries

### MEL builds (moat + trust surfaces)
- Mixed-network truth model (live/stale/historical/imported/partial/degraded/unsupported).
- Incident intelligence, evidence chains, and inspect-next guidance.
- Proofpack assembly + export with explicit evidence gaps.
- Action-outcome memory and before/after snapshots.
- Privacy/trust policy enforcement and degraded-state semantics.
- Local intelligence orchestration policy (task class routing, fallback reasoning).

### MEL borrows (mature OSS infrastructure)
- Crypto primitives: libsignal-compatible provider boundary.
- Event bus: NATS + JetStream.
- Blob/object storage: S3-compatible (MinIO in self-host examples).
- TURN/STUN relay: coturn.
- Local speech-to-text: whisper.cpp class runtime.
- Local inference runtime: Ollama (default), llama.cpp (advanced).

## Canonical architecture layers
1. **Operator layer**: API/CLI/UI and proofpack export.
2. **Truth kernel**: deterministic ingest truth + control lifecycle state.
3. **Incident memory**: evidence chain, incidents, action history, replay snapshots.
4. **Assistive intelligence**: optional local LLM summarization/drafting/comparison.
5. **Infrastructure adapters**: provider interfaces for bus/blob/relay/crypto/stt/inference.
6. **Self-host runtime**: MEL process + optional OSS sidecars.

## Local-first and privacy model
- Self-hosted mode is the only supported platform mode in base config.
- Telemetry defaults to disabled with explicit opt-in gate.
- Evidence export stays enabled by policy default.
- No silent cloud fallback in provider selection.
- Key custody is local deployment responsibility; MEL never requires managed key SaaS.

## Encryption baseline
- Use established cryptographic libraries behind `CryptoProvider`; do not implement custom ratchets/ciphers.
- Transport encryption requirements stay explicit in config and runtime health.
- Claims are bounded to proven behavior (configured provider + connectivity + persisted evidence).

## Local intelligence strategy (non-canonical)
- Deterministic truth remains canonical.
- Local inference is optional assistive output only.
- Task-aware routing chooses runtime/mode/hardware/compression per task class.
- Runtime can degrade to queued/partial/unavailable without blocking core MEL behavior.

## Deployment topologies
- **Core mode**: MEL only (no API keys, no LLM, no optional sidecars).
- **Comms-hub mode**: MEL + NATS + MinIO + coturn.
- **Assist mode**: add Ollama and/or llama.cpp sidecars.
- **Mixed mode**: task-aware routing across Ollama/llama.cpp with queue handoff.

## Current vs future vs unsupported
- Current: serial/TCP + MQTT ingest, deterministic truth/state APIs, incident and proofpack primitives.
- Foundation scaffolding (this pass): provider interfaces, runtime policy, self-host compose wiring, config policy fields.
- Future: production-grade provider implementations, queue workers, full media path integration.
- Explicitly unsupported: BLE/HTTP ingest as production-supported transports; MEL as RF routing stack.
