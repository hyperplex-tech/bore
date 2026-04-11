#!/usr/bin/env bash
#
# Backup bore configuration and state.
#
# Usage:
#   backup-config.sh [destination_dir]
#
# Backs up:
#   - ~/.config/bore/tunnels.yaml   (tunnel configuration)
#   - ~/.local/share/bore/state.db  (runtime state database)
#
# The backup is a timestamped .tar.gz archive placed in the destination
# directory (defaults to the current directory).

set -euo pipefail

CONFIG_DIR="${XDG_CONFIG_HOME:-$HOME/.config}/bore"
DATA_DIR="${XDG_DATA_HOME:-$HOME/.local/share}/bore"
DEST_DIR="${1:-.}"
TIMESTAMP="$(date +%Y%m%d_%H%M%S)"
ARCHIVE_NAME="bore-backup-${TIMESTAMP}.tar.gz"

if [ ! -d "$CONFIG_DIR" ] && [ ! -d "$DATA_DIR" ]; then
    echo "Error: No bore config or data directories found." >&2
    exit 1
fi

mkdir -p "$DEST_DIR"

# Collect files that exist
files=()
for f in "$CONFIG_DIR/tunnels.yaml" "$DATA_DIR/state.db"; do
    if [ -f "$f" ]; then
        files+=("$f")
    fi
done

if [ ${#files[@]} -eq 0 ]; then
    echo "Error: No config or state files found to back up." >&2
    exit 1
fi

tar -czf "${DEST_DIR}/${ARCHIVE_NAME}" "${files[@]}"

echo "Backup created: ${DEST_DIR}/${ARCHIVE_NAME}"
echo "Contents:"
tar -tzf "${DEST_DIR}/${ARCHIVE_NAME}"
