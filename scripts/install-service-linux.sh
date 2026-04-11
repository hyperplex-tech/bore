#!/usr/bin/env bash
#
# install-service.sh — Install bored as a systemd user service.
#
# Usage:
#   ./scripts/install-service.sh [path-to-bored-binary]
#
# This installs a systemd **user** service (not system-wide), so it runs
# as your user and has access to your SSH agent, keys, and config.
#
# After install:
#   systemctl --user start  bored     # start the daemon
#   systemctl --user stop   bored     # stop it
#   systemctl --user enable bored     # auto-start on login
#   systemctl --user disable bored    # disable auto-start
#   journalctl --user -u bored -f     # view logs
#
set -euo pipefail

BORED_BIN="${1:-}"
INSTALL_DIR="$HOME/.local/bin"
SERVICE_DIR="$HOME/.config/systemd/user"
SERVICE_NAME="bored"

# --- Resolve binary path ---

if [ -z "$BORED_BIN" ]; then
    # Try common locations.
    for candidate in ./bin/bored "$INSTALL_DIR/bored" "$(command -v bored 2>/dev/null || true)"; do
        if [ -n "$candidate" ] && [ -x "$candidate" ]; then
            BORED_BIN="$candidate"
            break
        fi
    done
fi

if [ -z "$BORED_BIN" ] || [ ! -f "$BORED_BIN" ]; then
    echo "Error: bored binary not found."
    echo "Usage: $0 [path-to-bored-binary]"
    echo ""
    echo "Build it first with: make build-daemon"
    exit 1
fi

BORED_BIN="$(realpath "$BORED_BIN")"
echo "Using binary: $BORED_BIN"

# --- Install binary ---

mkdir -p "$INSTALL_DIR"
if [ "$BORED_BIN" != "$INSTALL_DIR/bored" ]; then
    cp "$BORED_BIN" "$INSTALL_DIR/bored"
    chmod +x "$INSTALL_DIR/bored"
    echo "Installed binary to $INSTALL_DIR/bored"
else
    echo "Binary already at $INSTALL_DIR/bored"
fi

# --- Ensure XDG directories exist ---

mkdir -p "$HOME/.config/bore"
mkdir -p "$HOME/.local/share/bore"

# --- Create systemd unit ---

mkdir -p "$SERVICE_DIR"
cat > "$SERVICE_DIR/${SERVICE_NAME}.service" << EOF
[Unit]
Description=Bore SSH Tunnel Manager Daemon
Documentation=https://github.com/hyperplex-tech/bore
After=network.target

[Service]
Type=simple
ExecStart=$INSTALL_DIR/bored
Restart=on-failure
RestartSec=5

# Environment — these use XDG defaults, override if needed.
# Environment=BORE_CONFIG=%h/.config/bore/tunnels.yaml
# Environment=BORE_SOCKET=%h/.local/share/bore/bored.sock
# Environment=BORE_LOG_LEVEL=info

# Pass the SSH agent socket so the daemon can authenticate.
# The daemon also auto-discovers common socket paths, but setting this
# ensures it works even if the socket is in a non-standard location.
# %t expands to XDG_RUNTIME_DIR (e.g. /run/user/1000).
# GNOME Keyring:
Environment=SSH_AUTH_SOCK=%t/keyring/ssh

[Install]
WantedBy=default.target
EOF

echo "Created service: $SERVICE_DIR/${SERVICE_NAME}.service"

# --- Reload systemd ---

systemctl --user daemon-reload
echo "Reloaded systemd user daemon."

# --- Print instructions ---

echo ""
echo "============================================"
echo "  bored systemd service installed!"
echo "============================================"
echo ""
echo "  Start:    systemctl --user start  bored"
echo "  Stop:     systemctl --user stop   bored"
echo "  Enable:   systemctl --user enable bored"
echo "  Disable:  systemctl --user disable bored"
echo "  Status:   systemctl --user status bored"
echo "  Logs:     journalctl --user -u bored -f"
echo ""
echo "  Config:   ~/.config/bore/tunnels.yaml"
echo "  Socket:   ~/.local/share/bore/bored.sock"
echo "  Data:     ~/.local/share/bore/"
echo ""
echo "  Tip: Run 'systemctl --user enable bored' to"
echo "  auto-start the daemon when you log in."
echo ""
