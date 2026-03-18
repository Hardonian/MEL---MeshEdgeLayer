# Install on Termux

Termux support in RC1 is foreground/manual only. MEL does not claim durable background persistence across all Android devices.

```bash
make build
./scripts/termux-run.sh
```

If you use `termux-services` or `termux-boot`, treat that as device-specific operator work rather than a MEL guarantee.
