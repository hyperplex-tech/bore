#!/usr/bin/env bash
#
# build-dmg.sh — Package Bore.app into a distributable .dmg installer.
#
# Usage:
#   ./scripts/build-dmg.sh [version]
#
# Prerequisites:
#   - macOS (uses hdiutil)
#   - Bore.app already built (make build-desktop)
#   - CLI + daemon + TUI binaries built (make build)
#
# Output:
#   bin/Bore-<version>.dmg
#
set -euo pipefail

VERSION="${1:-$(git describe --tags --always --dirty 2>/dev/null || echo "dev")}"
BIN_DIR="bin"
DESKTOP_BIN="$BIN_DIR/bore-desktop"
DAEMON_BIN="$BIN_DIR/bored"
CLI_BIN="$BIN_DIR/bore"
TUI_BIN="$BIN_DIR/bore-tui"
DMG_NAME="Bore-${VERSION}.dmg"
DMG_PATH="$BIN_DIR/$DMG_NAME"
STAGING_DIR="$BIN_DIR/dmg-staging"
APP_DIR="$STAGING_DIR/Bore.app"
VOLUME_NAME="Bore $VERSION"

# --- Verify binaries exist ---

for bin in "$DESKTOP_BIN" "$DAEMON_BIN" "$CLI_BIN" "$TUI_BIN"; do
    if [ ! -f "$bin" ]; then
        echo "Error: $bin not found. Run 'make build' first."
        exit 1
    fi
done

echo "Building $DMG_NAME..."

# --- Clean previous staging ---

rm -rf "$STAGING_DIR"
rm -f "$DMG_PATH"

# --- Create .app bundle ---

mkdir -p "$APP_DIR/Contents/MacOS"
mkdir -p "$APP_DIR/Contents/Resources"

# Main executable
cp "$DESKTOP_BIN" "$APP_DIR/Contents/MacOS/bore-desktop"
chmod +x "$APP_DIR/Contents/MacOS/bore-desktop"

# Bundle CLI tools alongside the app so the installer can place them
cp "$DAEMON_BIN" "$APP_DIR/Contents/MacOS/bored"
cp "$CLI_BIN" "$APP_DIR/Contents/MacOS/bore"
cp "$TUI_BIN" "$APP_DIR/Contents/MacOS/bore-tui"
chmod +x "$APP_DIR/Contents/MacOS/bored" "$APP_DIR/Contents/MacOS/bore" "$APP_DIR/Contents/MacOS/bore-tui"

# Info.plist
sed "s/1.0.0/$VERSION/g" desktop/Info.plist > "$APP_DIR/Contents/Info.plist"

# Icon — generate .icns from the iconset if iconutil is available (macOS)
if [ -d assets/bore-desktop.iconset ]; then
    if command -v iconutil >/dev/null 2>&1; then
        iconutil -c icns -o "$APP_DIR/Contents/Resources/bore-desktop.icns" assets/bore-desktop.iconset
        echo "Generated .icns icon from iconset."
    else
        echo "Warning: iconutil not found — .app will use default icon."
        echo "  (iconutil is only available on macOS)"
    fi
elif [ -f assets/bore-desktop.icns ]; then
    cp assets/bore-desktop.icns "$APP_DIR/Contents/Resources/bore-desktop.icns"
fi

# --- Install script (bundled inside .app for Gatekeeper compatibility) ---

cat > "$APP_DIR/Contents/MacOS/install-cli" << 'SCRIPT'
#!/usr/bin/env bash
#
# Installs bore CLI tools and registers the daemon as a launchd agent.
#
set -euo pipefail

APP_PATH="/Applications/Bore.app"
INSTALL_DIR="$HOME/.local/bin"

if [ ! -d "$APP_PATH" ]; then
    APP_PATH="$HOME/Applications/Bore.app"
fi

if [ ! -d "$APP_PATH" ]; then
    echo "Error: Bore.app not found in /Applications or ~/Applications."
    echo "Please drag Bore.app to Applications first."
    exit 1
fi

echo "Installing bore CLI tools..."
echo ""

mkdir -p "$INSTALL_DIR"

for tool in bore bored bore-tui; do
    src="$APP_PATH/Contents/MacOS/$tool"
    if [ -f "$src" ]; then
        cp "$src" "$INSTALL_DIR/$tool"
        chmod +x "$INSTALL_DIR/$tool"
        echo "  Installed $tool to $INSTALL_DIR/"
    fi
done

# --- Register launchd agent for the daemon ---

PLIST_DIR="$HOME/Library/LaunchAgents"
LABEL="com.bore.daemon"
LOG_DIR="$HOME/Library/Logs/bore"

mkdir -p "$HOME/Library/Application Support/bore"
mkdir -p "$LOG_DIR"
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

launchctl load -w "$PLIST_DIR/${LABEL}.plist" 2>/dev/null || true

echo ""
echo "Done! The bore daemon is running."
echo ""

# Check PATH
if ! echo "$PATH" | tr ':' '\n' | grep -qx "$INSTALL_DIR"; then
    echo "Add ~/.local/bin to your PATH:"
    echo "  export PATH=\"\$HOME/.local/bin:\$PATH\""
    echo ""
    echo "Add that line to your ~/.zshrc or ~/.bashrc to make it permanent."
    echo ""
fi

echo "You can now use:"
echo "  bore status       — check daemon status"
echo "  bore-tui          — terminal UI"
echo "  Bore.app          — desktop GUI (in Applications)"
echo ""
SCRIPT
chmod +x "$APP_DIR/Contents/MacOS/install-cli"

# --- README with install instructions ---

cat > "$STAGING_DIR/README.txt" << 'README'
=====================================
  Bore - SSH Tunnel Manager
=====================================

  STEP 1:  Drag "Bore.app" into the "Applications" folder.

  STEP 2:  Open Bore.app from your Applications folder
           or search "Bore" in Spotlight.

  STEP 3 (optional):  To also install the CLI tools and
           background daemon, open Terminal and run:

           /Applications/Bore.app/Contents/MacOS/install-cli

-------------------------------------
  After installation:
-------------------------------------

  - Open "Bore" from Spotlight or your Applications folder.
  - Or use the terminal:
      bore status       Check daemon status
      bore-tui          Terminal UI
      bore add          Add a new tunnel

-------------------------------------
  To uninstall:
-------------------------------------

  Run in Terminal:
    /bin/bash -c "$(curl -fsSL https://raw.githubusercontent.com/hyperplex-tech/bore/main/scripts/uninstall-macos.sh)"

  Or download and run scripts/uninstall-macos.sh manually.

README

# --- Symlink to /Applications for drag-and-drop ---

ln -s /Applications "$STAGING_DIR/Applications"

# --- Create DMG ---

hdiutil create \
    -volname "$VOLUME_NAME" \
    -srcfolder "$STAGING_DIR" \
    -ov \
    -format UDZO \
    "$DMG_PATH"

# --- Cleanup ---

rm -rf "$STAGING_DIR"

echo ""
echo "============================================"
echo "  $DMG_NAME created!"
echo "============================================"
echo ""
echo "  Output: $DMG_PATH"
echo "  Size:   $(du -h "$DMG_PATH" | cut -f1)"
echo ""
echo "  Contents:"
echo "    Bore.app              — Desktop GUI + bundled CLI tools"
echo "    Applications symlink  — Drag Bore.app here to install"
echo "    Install CLI Tools     — Sets up CLI + daemon service"
echo ""
