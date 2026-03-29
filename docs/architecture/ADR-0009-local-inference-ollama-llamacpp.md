# ADR-0009-local-inference-ollama-llamacpp: Local inference strategy uses Ollama default + llama.cpp advanced path

## Status
Accepted

## Decision
Keep inference optional and non-canonical; route tasks by runtime policy across Ollama/llama.cpp with compression strategy controls.

## Consequences
- Reduces lock-in and recurring spend for core MEL operation.
- Keeps operator-truth claims bounded to locally verifiable evidence.
- Preserves swap-friendly provider seams for future implementation changes.
