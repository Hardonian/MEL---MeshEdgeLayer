# MEL Operations Runbook

## Daily Operations

### Morning Health
```bash
./bin/mel doctor --config config.json
./bin/mel status --config config.json
```

### Monitoring
- Embedded UI: `/status`
- Diagnostics: `/api/v1/diagnostics`

## Common Issues

### Transport Not Connecting
1. Check port: `ls /dev/tty*`
2. Check permissions: `dialout` group
3. Run: `./bin/mel doctor`

### Database Locked
1. Check for other processes
2. Kill stale processes: `./bin/mel kill-stale`
3. Restart: `./bin/mel serve`

### Control Actions Not Executing
1. Check queue: `./bin/mel actions pending`
2. Verify approval workflow
3. Check capabilities on API key

## Maintenance

### Backup
```bash
./bin/mel backup --output backup.tar.gz
```

### Update
```bash
make update
./bin/mel migrate
```

### Cleanup
```bash
./bin/mel prune --older-than 30d
```

## Metrics

Tracked:
- Uptime per transport
- Packet throughput
- Action latency
- Error rates
