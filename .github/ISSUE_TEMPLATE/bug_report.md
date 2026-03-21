---
name: Bug Report
about: Create a report to help us improve MEL
title: '[BUG] '
labels: bug
assignees: ''
---

## Description

A clear and concise description of what the bug is.

## Environment

- MEL Version: (e.g., 0.1.0-rc1)
- Host OS: (e.g., Ubuntu 22.04, Windows 11)
- Go Version: (if building from source)
- Transport Type: (Serial / TCP / MQTT)

## Reproduction Steps

1.
2.
3.

## Expected Behavior

A clear and concise description of what you expected to happen.

## Actual Behavior

What actually happened? Include any relevant error messages.

## Config Posture

Please provide your `mel.json` config (redact sensitive keys like MQTT passwords or precise locations).

```json
// Paste config here
```

## Logs & Doctor Output

Please run `mel doctor --config <path>` and paste the output.

```text
// Paste mel doctor output here
```

## Additional Context

Add any other context about the problem here (e.g., node model, radio firmware version).
