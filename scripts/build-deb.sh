#!/usr/bin/env bash
#
# build-deb.sh — Build a .deb package for Bore.
#
# Usage:
#   ./scripts/build-deb.sh [version] [arch]
#
# Prerequisites:
#   - dpkg-deb (installed by default on Debian/Ubuntu)
#   - Built binaries in bin/ (make build)
#
# Output:
#   bin/bore_<version>_<arch>.deb
#
set -euo pipefail

VERSION="${1:-$(git describe --tags --always --dirty 2>/dev/null || echo "0.0.0")}"
# Strip leading "v" if present (e.g. v1.2.3 -> 1.2.3) — deb versions must not start with a letter.
VERSION="${VERSION#v}"
ARCH="${2:-$(dpkg --print-architecture 2>/dev/null || echo "amd64")}"

BIN_DIR="bin"
DEB_NAME="bore_${VERSION}_${ARCH}.deb"
DEB_PATH="$BIN_DIR/$DEB_NAME"
STAGING="$BIN_DIR/deb-staging"

DAEMON_BIN="$BIN_DIR/bored"
CLI_BIN="$BIN_DIR/bore"
TUI_BIN="$BIN_DIR/bore-tui"
DESKTOP_BIN="$BIN_DIR/bore-desktop"

# --- Verify binaries ---

for bin in "$DAEMON_BIN" "$CLI_BIN" "$TUI_BIN"; do
    if [ ! -f "$bin" ]; then
        echo "Error: $bin not found. Run 'make build' first."
        exit 1
    fi
done

echo "Building $DEB_NAME..."

# --- Clean ---

rm -rf "$STAGING"
rm -f "$DEB_PATH"

# --- Directory structure ---

mkdir -p "$STAGING/DEBIAN"
mkdir -p "$STAGING/usr/bin"
mkdir -p "$STAGING/usr/lib/systemd/user"
mkdir -p "$STAGING/usr/share/applications"
mkdir -p "$STAGING/usr/share/doc/bore"

# --- Binaries ---

cp "$CLI_BIN"    "$STAGING/usr/bin/bore"
cp "$DAEMON_BIN" "$STAGING/usr/bin/bored"
cp "$TUI_BIN"    "$STAGING/usr/bin/bore-tui"
chmod 755 "$STAGING/usr/bin/bore" "$STAGING/usr/bin/bored" "$STAGING/usr/bin/bore-tui"

if [ -f "$DESKTOP_BIN" ]; then
    cp "$DESKTOP_BIN" "$STAGING/usr/bin/bore-desktop"
    chmod 755 "$STAGING/usr/bin/bore-desktop"
fi

# --- Icons ---

for size in 16 24 32 48 64 128 256 512; do
    icon_file="assets/icon-${size}x${size}.png"
    if [ -f "$icon_file" ]; then
        dest="$STAGING/usr/share/icons/hicolor/${size}x${size}/apps"
        mkdir -p "$dest"
        cp "$icon_file" "$dest/bore.png"
    fi
done

# --- Desktop entry ---
# The .desktop file Exec= must point to /usr/bin/ for system installs.

cat > "$STAGING/usr/share/applications/bore-desktop.desktop" << 'EOF'
[Desktop Entry]
Name=Bore
Comment=SSH Tunnel Manager
Exec=bore-desktop
Icon=bore
Terminal=false
Type=Application
Categories=Network;Utility;
Keywords=ssh;tunnel;port;forward;
StartupWMClass=bore-desktop
EOF

# --- Systemd user service ---

cat > "$STAGING/usr/lib/systemd/user/bored.service" << 'EOF'
[Unit]
Description=Bore SSH Tunnel Manager Daemon
Documentation=https://github.com/hyperplex-tech/bore
After=network.target

[Service]
Type=simple
ExecStart=/usr/bin/bored
Restart=on-failure
RestartSec=5

# Pass the SSH agent socket so the daemon can authenticate.
# %t expands to XDG_RUNTIME_DIR (e.g. /run/user/1000).
Environment=SSH_AUTH_SOCK=%t/keyring/ssh

[Install]
WantedBy=default.target
EOF

# --- Copyright / docs ---

cat > "$STAGING/usr/share/doc/bore/copyright" << EOF
Format: https://www.debian.org/doc/packaging-manuals/copyright-format/1.0/
Upstream-Name: bore
Source: https://github.com/hyperplex-tech/bore

Files: *
Copyright: $(date +%Y) Hyperplex
License: MIT
EOF

# --- DEBIAN/control ---

# Calculate installed size in KB.
INSTALLED_SIZE=$(du -sk "$STAGING" | cut -f1)

cat > "$STAGING/DEBIAN/control" << EOF
Package: bore
Version: $VERSION
Section: net
Priority: optional
Architecture: $ARCH
Installed-Size: $INSTALLED_SIZE
Maintainer: Hyperplex <dev@hyperplex.tech>
Homepage: https://github.com/hyperplex-tech/bore
Description: SSH tunnel manager with GUI, TUI, and CLI
 Bore manages persistent SSH tunnels with automatic reconnection,
 a background daemon, and multiple interfaces: desktop GUI (Wails),
 terminal UI (bubbletea), and command-line. Supports local, remote,
 dynamic (SOCKS5), and Kubernetes port-forward tunnels.
EOF

# --- DEBIAN/postinst ---

cat > "$STAGING/DEBIAN/postinst" << 'EOF'
#!/bin/sh
set -e

# Refresh desktop database so the app appears in launchers.
if command -v update-desktop-database >/dev/null 2>&1; then
    update-desktop-database /usr/share/applications 2>/dev/null || true
fi

# Refresh icon cache.
if command -v gtk-update-icon-cache >/dev/null 2>&1; then
    gtk-update-icon-cache -f -t /usr/share/icons/hicolor 2>/dev/null || true
fi

echo ""
echo "Bore installed!"
echo ""
echo "  Enable the daemon for your user:"
echo "    systemctl --user enable --now bored"
echo ""
echo "  Then use:"
echo "    bore status       — check daemon status"
echo "    bore-tui          — terminal UI"
echo "    bore-desktop      — desktop GUI (also in your app launcher)"
echo ""
EOF
chmod 755 "$STAGING/DEBIAN/postinst"

# --- DEBIAN/prerm ---

cat > "$STAGING/DEBIAN/prerm" << 'EOF'
#!/bin/sh
set -e

# Best-effort stop for all users who might have the service running.
# This runs as root during package removal, but the service is per-user,
# so we just print a reminder.
echo ""
echo "Note: The bored systemd user service is not automatically stopped."
echo "  Each user who enabled it should run:"
echo "    systemctl --user disable --now bored"
echo ""
EOF
chmod 755 "$STAGING/DEBIAN/prerm"

# --- DEBIAN/postrm ---

cat > "$STAGING/DEBIAN/postrm" << 'EOF'
#!/bin/sh
set -e

if command -v update-desktop-database >/dev/null 2>&1; then
    update-desktop-database /usr/share/applications 2>/dev/null || true
fi
if command -v gtk-update-icon-cache >/dev/null 2>&1; then
    gtk-update-icon-cache -f -t /usr/share/icons/hicolor 2>/dev/null || true
fi
EOF
chmod 755 "$STAGING/DEBIAN/postrm"

# --- Build .deb ---

dpkg-deb --build --root-owner-group "$STAGING" "$DEB_PATH"

# --- Cleanup ---

rm -rf "$STAGING"

echo ""
echo "============================================"
echo "  $DEB_NAME created!"
echo "============================================"
echo ""
echo "  Output: $DEB_PATH"
echo "  Size:   $(du -h "$DEB_PATH" | cut -f1)"
echo ""
echo "  Install:  sudo dpkg -i $DEB_PATH"
echo "  Remove:   sudo apt remove bore"
echo "  Purge:    sudo apt purge bore"
echo ""
