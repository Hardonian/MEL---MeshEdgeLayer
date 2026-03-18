#!/usr/bin/env bash
set -euo pipefail
PREFIX="${PREFIX:-/usr/local}"
CONFIG_DIR="${CONFIG_DIR:-/etc/mel}"
DATA_DIR="${DATA_DIR:-/var/lib/mel}"
SYSTEMD_DIR="${SYSTEMD_DIR:-/etc/systemd/system}"
rm -f "$PREFIX/bin/mel" "$SYSTEMD_DIR/mel.service"
echo "Binary and service removed."
echo "Config preserved at $CONFIG_DIR"
echo "Data preserved at $DATA_DIR"
echo "Remove them manually if you want a full uninstall."
