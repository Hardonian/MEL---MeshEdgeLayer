# ADR-0005-blob-store-s3-compatible: Object storage uses S3-compatible API

## Status
Accepted

## Decision
Use S3-compatible storage (for example MinIO self-hosted) for proofpack attachments and large evidence blobs.

## Consequences
- Reduces lock-in and recurring spend for core MEL operation.
- Keeps operator-truth claims bounded to locally verifiable evidence.
- Preserves swap-friendly provider seams for future implementation changes.
