#!/usr/bin/env bash
#
# uninstall-service.sh — Remove the bored systemd user service.
#
set -euo pipefail

SERVICE_NAME="bored"
SERVICE_DIR="$HOME/.config/systemd/user"
INSTALL_DIR="$HOME/.local/bin"

echo "Stopping and disabling bored service..."
systemctl --user stop "$SERVICE_NAME" 2>/dev/null || true
systemctl --user disable "$SERVICE_NAME" 2>/dev/null || true

if [ -f "$SERVICE_DIR/${SERVICE_NAME}.service" ]; then
    rm "$SERVICE_DIR/${SERVICE_NAME}.service"
    echo "Removed $SERVICE_DIR/${SERVICE_NAME}.service"
fi

systemctl --user daemon-reload

echo "Service removed. Config (~/.config/bore/) and data (~/.local/share/bore/) were NOT removed."
echo "Delete them manually if you want a clean uninstall."
