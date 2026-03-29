# ADR-0003-crypto-provider-libsignal-boundary: Crypto provider boundary uses libsignal-compatible primitives

## Status
Accepted

## Decision
Adopt a CryptoProvider abstraction and require mature OSS crypto implementations for sessions/envelopes instead of custom cryptography.

## Consequences
- Reduces lock-in and recurring spend for core MEL operation.
- Keeps operator-truth claims bounded to locally verifiable evidence.
- Preserves swap-friendly provider seams for future implementation changes.
