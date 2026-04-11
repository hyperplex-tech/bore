#!/usr/bin/env bash
#
# uninstall-service-macos.sh — Remove the bored launchd user agent.
#
set -euo pipefail

LABEL="com.bore.daemon"
PLIST="$HOME/Library/LaunchAgents/${LABEL}.plist"

echo "Stopping and unloading bored agent..."
launchctl stop "$LABEL" 2>/dev/null || true
launchctl unload "$PLIST" 2>/dev/null || true

if [ -f "$PLIST" ]; then
    rm "$PLIST"
    echo "Removed $PLIST"
fi

echo "Service removed. Config and data were NOT removed."
