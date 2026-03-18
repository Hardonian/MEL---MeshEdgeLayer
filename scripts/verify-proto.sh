#!/usr/bin/env bash
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
mkdir -p "$ROOT/internal/meshtastic/protobuf"
protoc -I "$ROOT/proto" --descriptor_set_out="$ROOT/internal/meshtastic/protobuf/meshtastic.pb" "$ROOT/proto/meshtastic/mqtt.proto"
