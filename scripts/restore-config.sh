#!/usr/bin/env bash
#
# Restore bore configuration and state from a backup archive.
#
# Usage:
#   restore-config.sh <backup_archive>
#
# Restores:
#   - ~/.config/bore/tunnels.yaml   (tunnel configuration)
#   - ~/.local/share/bore/state.db  (runtime state database)
#
# Existing files are backed up to *.bak before overwriting.

set -euo pipefail

if [ $# -eq 0 ]; then
    echo "Usage: restore-config.sh <backup_archive>" >&2
    echo "  e.g. restore-config.sh bore-backup-20260409_120000.tar.gz" >&2
    exit 1
fi

ARCHIVE="$1"

if [ ! -f "$ARCHIVE" ]; then
    echo "Error: Archive not found: $ARCHIVE" >&2
    exit 1
fi

CONFIG_DIR="${XDG_CONFIG_HOME:-$HOME/.config}/bore"
DATA_DIR="${XDG_DATA_HOME:-$HOME/.local/share}/bore"

echo "Archive contents:"
tar -tzf "$ARCHIVE"
echo ""

# Back up existing files before overwriting
for f in "$CONFIG_DIR/tunnels.yaml" "$DATA_DIR/state.db"; do
    if [ -f "$f" ]; then
        cp "$f" "$f.bak"
        echo "Backed up existing $f -> $f.bak"
    fi
done

# Ensure target directories exist
mkdir -p "$CONFIG_DIR"
mkdir -p "$DATA_DIR"

# Restore — extract with absolute paths as stored in the archive
tar -xzf "$ARCHIVE" -C /

echo ""
echo "Restore complete."
echo "Restored files:"
[ -f "$CONFIG_DIR/tunnels.yaml" ] && echo "  $CONFIG_DIR/tunnels.yaml"
[ -f "$DATA_DIR/state.db" ]       && echo "  $DATA_DIR/state.db"
