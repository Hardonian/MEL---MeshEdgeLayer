# ADR-0007-local-stt-whispercpp: Local speech-to-text baseline is whisper.cpp class runtime

## Status
Accepted

## Decision
Speech-to-text is optional and local-first with whisper.cpp-compatible adapters, no paid API dependency required.

## Consequences
- Reduces lock-in and recurring spend for core MEL operation.
- Keeps operator-truth claims bounded to locally verifiable evidence.
- Preserves swap-friendly provider seams for future implementation changes.
