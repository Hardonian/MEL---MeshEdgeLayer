# QUICKSTART

## Day 0 (10–15 minutes)

```bash
make build
./bin/mel init --config .tmp/mel.json
./bin/mel doctor --config .tmp/mel.json
./bin/mel serve --config .tmp/mel.json
```

Open <http://127.0.0.1:8080> and verify:

- status page loads
- transport state is explicit (not implicitly healthy)
- incident queue is visible even if empty

## Next reads

- [First hour guide](./FIRST_HOUR_GUIDE.md)
- [First incident guide](./FIRST_INCIDENT_GUIDE.md)
- [Common misreads](./COMMON_MISREADS.md)
