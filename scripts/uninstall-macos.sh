#!/usr/bin/env bash
#
# uninstall-macos.sh — Fully uninstall bore from macOS (.dmg or make install).
#
# Usage:
#   ./scripts/uninstall-macos.sh [--purge]
#
# Without --purge: removes binaries, app, and daemon service.
# With    --purge: also removes config, data, and logs.
#
set -euo pipefail

PURGE=false
if [ "${1:-}" = "--purge" ]; then
    PURGE=true
fi

echo "=== Uninstalling bore from macOS ==="
echo ""

# --- Stop and remove launchd agent ---

LABEL="com.bore.daemon"
PLIST="$HOME/Library/LaunchAgents/${LABEL}.plist"

echo "Stopping daemon..."
launchctl stop "$LABEL" 2>/dev/null || true
launchctl unload "$PLIST" 2>/dev/null || true

if [ -f "$PLIST" ]; then
    rm "$PLIST"
    echo "  Removed $PLIST"
fi

# --- Remove binaries ---

INSTALL_DIR="$HOME/.local/bin"
for tool in bore bored bore-tui bore-desktop; do
    if [ -f "$INSTALL_DIR/$tool" ]; then
        rm "$INSTALL_DIR/$tool"
        echo "  Removed $INSTALL_DIR/$tool"
    fi
done

# --- Remove .app bundle ---

for app_dir in "/Applications/Bore.app" "$HOME/Applications/Bore.app"; do
    if [ -d "$app_dir" ]; then
        rm -rf "$app_dir"
        echo "  Removed $app_dir"
    fi
done

echo ""
echo "Removed: binaries, desktop app, and daemon service."

# --- Purge config, data, and logs ---

if [ "$PURGE" = true ]; then
    echo ""
    rm -rf "$HOME/Library/Application Support/bore"
    echo "  Removed ~/Library/Application Support/bore/"
    rm -rf "$HOME/Library/Logs/bore"
    echo "  Removed ~/Library/Logs/bore/"
    echo ""
    echo "bore is fully removed from this system."
else
    echo ""
    echo "Config and data were NOT removed:"
    echo "  ~/Library/Application Support/bore/"
    echo "  ~/Library/Logs/bore/"
    echo ""
    echo "To also remove config and data, re-run with --purge:"
    echo "  ./scripts/uninstall-macos.sh --purge"
fi
