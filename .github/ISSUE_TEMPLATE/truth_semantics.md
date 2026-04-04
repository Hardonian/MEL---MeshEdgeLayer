---
name: Truth / semantics / degraded state
about: Incorrect or ambiguous live vs stale, evidence posture, transport, topology, or control semantics
title: '[TRUTH] '
labels: bug
assignees: ''
---

## Surface

- [ ] Web console
- [ ] CLI (`mel doctor`, `mel status`, etc.)
- [ ] API response shape / field semantics
- [ ] Documentation wording

## What the UI or output claimed

Quote labels, JSON fields, or doc text.

## What evidence you have

- Timestamps, ingest source, transport type
- Whether data was imported, replayed, or historical-only
- Relevant incident or node IDs (non-secret)

## Why this breaks operator trust

(e.g. degraded state looks healthy; topology implies a path that ingest does not prove; control phase ambiguous)

## Suggested correct behavior (optional)

Tie suggestions to **persisted evidence** or explicit unknown/degraded signaling — not visual inference alone.
