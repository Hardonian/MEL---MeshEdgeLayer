#!/usr/bin/env bash
set -euo pipefail
PREFIX="${PREFIX:-/usr/local}"
CONFIG_PATH="${CONFIG_PATH:-/etc/mel/mel.json}"
BACKUP_DIR="${BACKUP_DIR:-/var/lib/mel/backups}"
NEW_BIN="${NEW_BIN:-$(pwd)/bin/mel}"
install -d "$BACKUP_DIR"
"$PREFIX/bin/mel" backup create --config "$CONFIG_PATH" --out "$BACKUP_DIR/pre-upgrade-$(date -u +%Y%m%dT%H%M%SZ).tgz"
install -m 0755 "$NEW_BIN" "$PREFIX/bin/mel"
echo "Upgrade complete. Restart with: systemctl restart mel"
