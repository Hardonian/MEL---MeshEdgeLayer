#!/data/data/com.termux/files/usr/bin/bash
set -euo pipefail
PREFIX_DIR="${PREFIX_DIR:-$HOME/.local/share/mel}"
CONFIG_PATH="${CONFIG_PATH:-$HOME/.config/mel/mel.json}"
mkdir -p "$PREFIX_DIR" "$(dirname "$CONFIG_PATH")"
if [[ ! -f "$CONFIG_PATH" ]]; then
  ./bin/mel init --config "$CONFIG_PATH"
fi
exec ./bin/mel serve --config "$CONFIG_PATH"
