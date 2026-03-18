#!/usr/bin/env bash
set -euo pipefail
PREFIX="${PREFIX:-/usr/local}"
CONFIG_DIR="${CONFIG_DIR:-/etc/mel}"
DATA_DIR="${DATA_DIR:-/var/lib/mel}"
SYSTEMD_DIR="${SYSTEMD_DIR:-/etc/systemd/system}"
BIN_SOURCE="${BIN_SOURCE:-$(pwd)/bin/mel}"
SERVICE_USER="${SERVICE_USER:-mel}"
SERVICE_GROUP="${SERVICE_GROUP:-$SERVICE_USER}"
SERIAL_GROUP="${SERIAL_GROUP:-dialout}"

install -d "$PREFIX/bin" "$CONFIG_DIR" "$DATA_DIR"
install -m 0755 "$BIN_SOURCE" "$PREFIX/bin/mel"
if command -v useradd >/dev/null 2>&1; then
  useradd --system --home "$DATA_DIR" --shell /usr/sbin/nologin "$SERVICE_USER" 2>/dev/null || true
fi
if command -v getent >/dev/null 2>&1 && getent group "$SERIAL_GROUP" >/dev/null; then
  usermod -aG "$SERVICE_GROUP","$SERIAL_GROUP" "$SERVICE_USER" 2>/dev/null || true
fi
if [[ ! -f "$CONFIG_DIR/mel.json" ]]; then
  "$PREFIX/bin/mel" init --config "$CONFIG_DIR/mel.json"
fi
chown root:"$SERVICE_GROUP" "$CONFIG_DIR/mel.json" 2>/dev/null || true
chmod 0640 "$CONFIG_DIR/mel.json" 2>/dev/null || true
chown -R "$SERVICE_USER":"$SERVICE_GROUP" "$DATA_DIR" 2>/dev/null || true
install -m 0644 docs/ops/systemd/mel.service "$SYSTEMD_DIR/mel.service"
sed -i "s#/usr/local#$PREFIX#g" "$SYSTEMD_DIR/mel.service"
sed -i "s#/etc/mel#$CONFIG_DIR#g" "$SYSTEMD_DIR/mel.service"
sed -i "s#/var/lib/mel#$DATA_DIR#g" "$SYSTEMD_DIR/mel.service"
sed -i "s#User=mel#User=$SERVICE_USER#g" "$SYSTEMD_DIR/mel.service"
sed -i "s#Group=mel#Group=$SERVICE_GROUP#g" "$SYSTEMD_DIR/mel.service"
echo "Installed MEL to $PREFIX/bin/mel"
echo "Config: $CONFIG_DIR/mel.json"
echo "Data: $DATA_DIR"
echo "Service account: $SERVICE_USER:$SERVICE_GROUP"
echo "Run: systemctl daemon-reload && systemctl enable --now mel"
