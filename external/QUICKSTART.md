# MEL Quick Start

## 10 Minutes

```bash
# Build
make build

# Init config
./bin/mel init --config my-mel.json

# Start
./bin/mel serve --config my-mel.json
```

## First Commands

```bash
# Check status
./bin/mel status --config my-mel.json

# View transports
./bin/mel transports --config my-mel.json

# Run diagnostics
./bin/mel doctor --config my-mel.json
```

## Docker

```bash
docker run -v mel-data:/data hardonian/mel:latest \
  serve --config /data/mel.json
```

## Configuration

Minimal `config.json`:
```json
{
  "transport": {
    "enabled": ["serial"],
    "serial": { "port": "/dev/ttyUSB0" }
  }
}
```
