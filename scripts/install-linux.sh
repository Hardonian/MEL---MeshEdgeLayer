#!/usr/bin/env bash
set -euo pipefail
PREFIX="${PREFIX:-/usr/local}"
CONFIG_DIR="${CONFIG_DIR:-/etc/mel}"
DATA_DIR="${DATA_DIR:-/var/lib/mel}"
SYSTEMD_DIR="${SYSTEMD_DIR:-/etc/systemd/system}"
BIN_SOURCE="${BIN_SOURCE:-$(pwd)/bin/mel}"

install -d "$PREFIX/bin" "$CONFIG_DIR" "$DATA_DIR"
install -m 0755 "$BIN_SOURCE" "$PREFIX/bin/mel"
if [[ ! -f "$CONFIG_DIR/mel.json" ]]; then
  "$PREFIX/bin/mel" init --config "$CONFIG_DIR/mel.json"
fi
install -m 0644 docs/ops/systemd/mel.service "$SYSTEMD_DIR/mel.service"
sed -i "s#/usr/local#$PREFIX#g" "$SYSTEMD_DIR/mel.service"
sed -i "s#/etc/mel#$CONFIG_DIR#g" "$SYSTEMD_DIR/mel.service"
sed -i "s#/var/lib/mel#$DATA_DIR#g" "$SYSTEMD_DIR/mel.service"
echo "Installed MEL to $PREFIX/bin/mel"
echo "Config: $CONFIG_DIR/mel.json"
echo "Data: $DATA_DIR"
echo "Run: systemctl daemon-reload && systemctl enable --now mel"
