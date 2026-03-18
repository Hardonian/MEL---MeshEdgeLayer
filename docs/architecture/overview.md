# Architecture Overview

MEL v0.1 uses a single Go module with stdlib-only runtime dependencies. The daemon owns config loading, SQLite migrations, event publication, policy/privacy evaluation, transport lifecycle, and the local web server. A real MQTT transport ingests Meshtastic-style `ServiceEnvelope` payloads and persists truthful packet/node state.
