#!/usr/bin/env bash
#
# install-service-macos.sh — Install bored as a launchd user agent.
#
# Usage:
#   ./scripts/install-service-macos.sh [path-to-bored-binary]
#
# After install:
#   launchctl start  com.bore.daemon     # start the daemon
#   launchctl stop   com.bore.daemon     # stop it
#   launchctl unload ~/Library/LaunchAgents/com.bore.daemon.plist  # disable
#
set -euo pipefail

BORED_BIN="${1:-}"
INSTALL_DIR="$HOME/.local/bin"
PLIST_DIR="$HOME/Library/LaunchAgents"
LABEL="com.bore.daemon"
LOG_DIR="$HOME/Library/Logs/bore"

# --- Resolve binary path ---

if [ -z "$BORED_BIN" ]; then
    for candidate in ./bin/bored "$INSTALL_DIR/bored" /usr/local/bin/bored "$(command -v bored 2>/dev/null || true)"; do
        if [ -n "$candidate" ] && [ -x "$candidate" ]; then
            BORED_BIN="$candidate"
            break
        fi
    done
fi

if [ -z "$BORED_BIN" ] || [ ! -f "$BORED_BIN" ]; then
    echo "Error: bored binary not found."
    echo "Usage: $0 [path-to-bored-binary]"
    echo "Build it first with: make build-daemon"
    exit 1
fi

BORED_BIN="$(cd "$(dirname "$BORED_BIN")" && pwd)/$(basename "$BORED_BIN")"
echo "Using binary: $BORED_BIN"

# --- Install binary ---

mkdir -p "$INSTALL_DIR"
if [ "$BORED_BIN" != "$INSTALL_DIR/bored" ]; then
    cp "$BORED_BIN" "$INSTALL_DIR/bored"
    chmod +x "$INSTALL_DIR/bored"
    echo "Installed binary to $INSTALL_DIR/bored"
fi

# --- Ensure directories exist ---

mkdir -p "$HOME/Library/Application Support/bore"
mkdir -p "$LOG_DIR"

# --- Create launchd plist ---

mkdir -p "$PLIST_DIR"
cat > "$PLIST_DIR/${LABEL}.plist" << EOF
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN"
  "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>Label</key>
    <string>${LABEL}</string>
    <key>ProgramArguments</key>
    <array>
        <string>${INSTALL_DIR}/bored</string>
    </array>
    <key>RunAtLoad</key>
    <true/>
    <key>KeepAlive</key>
    <true/>
    <key>StandardOutPath</key>
    <string>${LOG_DIR}/bored.log</string>
    <key>StandardErrorPath</key>
    <string>${LOG_DIR}/bored.log</string>
</dict>
</plist>
EOF

echo "Created plist: $PLIST_DIR/${LABEL}.plist"

# --- Load the agent ---

launchctl load -w "$PLIST_DIR/${LABEL}.plist"

echo ""
echo "============================================"
echo "  bored launchd agent installed!"
echo "============================================"
echo ""
echo "  Start:    launchctl start  $LABEL"
echo "  Stop:     launchctl stop   $LABEL"
echo "  Unload:   launchctl unload ~/Library/LaunchAgents/${LABEL}.plist"
echo ""
echo "  Config:   ~/Library/Application Support/bore/tunnels.yaml"
echo "  Logs:     $LOG_DIR/bored.log"
echo ""
echo "  The daemon will auto-start on login."
echo ""
