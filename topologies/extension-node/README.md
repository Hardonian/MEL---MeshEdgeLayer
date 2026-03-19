# Extension node scaffold

The extension node is the constrained device boundary for field-side collection and forwarding.

Expected placement:

- `config/` for small per-device settings such as transport credentials, sampling thresholds, and OTA-safe runtime knobs.
- `memory-management/` for small bounded buffers or local error journals only when verified code needs them.

Current truth:

- This repository does not yet ship extension-node firmware or device-side runtimes.
- The scaffold exists so future work lands in deterministic locations without overstating implemented support.
