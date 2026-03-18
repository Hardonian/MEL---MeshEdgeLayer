# Troubleshooting

## Doctor fails on config permissions

Tighten the config file to `0600` or `0640` with a group readable by the MEL service account.

## UI shows no nodes

That means no real packets have been ingested yet. Check transport health, packet counters, and `last_successful_ingest`.

## `mel doctor` reports no transports enabled

This is an explicit idle state, not a silent failure. MEL will start, keep the UI/API available, and show empty node/message state until you enable a supported transport.

## Privacy audit is noisy

That is intentional for risky posture. Use the remediation text or document why the override is acceptable.
