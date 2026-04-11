.PHONY: all build build-daemon build-cli build-desktop build-frontend proto clean test test-unit test-integration lint release install uninstall check dev dev-desktop purge deb dmg windows-installer uninstall-macos uninstall-windows release-uninstall

# Platform detection
UNAME := $(shell uname -s 2>/dev/null || echo Windows)

# Binary output
BIN_DIR := bin
ifeq ($(OS),Windows_NT)
  EXT := .exe
else
  EXT :=
endif
DAEMON_BIN := $(BIN_DIR)/bored$(EXT)
CLI_BIN := $(BIN_DIR)/bore$(EXT)
DESKTOP_BIN := $(BIN_DIR)/bore-desktop$(EXT)
TUI_BIN := $(BIN_DIR)/bore-tui$(EXT)
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS := -ldflags "-s -w -X github.com/hyperplex-tech/bore/internal/version.Version=$(VERSION)"
# On Windows, build the desktop app as a GUI subsystem binary (no console window).
ifeq ($(OS),Windows_NT)
  DESKTOP_LDFLAGS := -ldflags "-s -w -H windowsgui -X github.com/hyperplex-tech/bore/internal/version.Version=$(VERSION)"
else
  DESKTOP_LDFLAGS := $(LDFLAGS)
endif

# Proto
PROTO_DIR := api/proto
GEN_DIR := gen

all: proto build

build: build-daemon build-cli build-tui build-desktop

build-daemon:
	@mkdir -p $(BIN_DIR)
	go build $(LDFLAGS) -o $(DAEMON_BIN) ./cmd/bored

build-cli:
	@mkdir -p $(BIN_DIR)
	go build $(LDFLAGS) -o $(CLI_BIN) ./cmd/bore

build-tui:
	@mkdir -p $(BIN_DIR)
	go build $(LDFLAGS) -o $(TUI_BIN) ./cmd/bore-tui

build-desktop: build-frontend
	@mkdir -p $(BIN_DIR)
	@TAGS="desktop,production"; \
	if [ "$(UNAME)" = "Linux" ]; then TAGS="desktop,production,webkit2_41"; fi; \
	go build $(DESKTOP_LDFLAGS) -tags "$$TAGS" -o $(DESKTOP_BIN) ./cmd/bore-desktop

build-frontend:
	cd desktop/frontend && npm install && npm run build

install: build
	@echo "=== Installing bore ==="
	@echo ""
ifeq ($(UNAME),Linux)
	@mkdir -p $(HOME)/.local/bin
	@mkdir -p $(HOME)/.config/bore
	@mkdir -p $(HOME)/.local/share/bore
	@cp $(DAEMON_BIN) $(HOME)/.local/bin/bored
	@cp $(CLI_BIN) $(HOME)/.local/bin/bore
	@cp $(TUI_BIN) $(HOME)/.local/bin/bore-tui
	@chmod +x $(HOME)/.local/bin/bored $(HOME)/.local/bin/bore $(HOME)/.local/bin/bore-tui
	@echo "Installed binaries to ~/.local/bin/"
	@# Desktop app integration
	@if [ -f $(DESKTOP_BIN) ]; then \
		cp $(DESKTOP_BIN) $(HOME)/.local/bin/bore-desktop; \
		chmod +x $(HOME)/.local/bin/bore-desktop; \
		mkdir -p $(HOME)/.local/share/applications; \
		cp desktop/bore-desktop.desktop $(HOME)/.local/share/applications/bore-desktop.desktop; \
		for size in 16 24 32 48 64 128 256 512; do \
			icon_file="assets/icon-$${size}x$${size}.png"; \
			if [ -f "$$icon_file" ]; then \
				dest="$(HOME)/.local/share/icons/hicolor/$${size}x$${size}/apps"; \
				mkdir -p "$$dest"; \
				cp "$$icon_file" "$$dest/bore.png"; \
			fi; \
		done; \
		if command -v update-desktop-database >/dev/null 2>&1; then \
			update-desktop-database $(HOME)/.local/share/applications 2>/dev/null || true; \
		fi; \
		if command -v gtk-update-icon-cache >/dev/null 2>&1; then \
			gtk-update-icon-cache -f -t $(HOME)/.local/share/icons/hicolor 2>/dev/null || true; \
		fi; \
		echo "Installed desktop app — search 'Bore' in your app launcher."; \
	fi
	@if command -v systemctl >/dev/null 2>&1; then \
		./scripts/install-service-linux.sh $(DAEMON_BIN); \
		systemctl --user enable --now bored; \
		echo ""; \
		echo "Daemon is running. You can now use:"; \
		echo "  bore status       — check daemon status"; \
		echo "  bore-tui          — terminal UI"; \
	else \
		echo "systemd not available — start the daemon manually:"; \
		echo "  bored &"; \
	fi
	@echo ""
	@if echo "$$PATH" | tr ':' '\n' | grep -qx "$$HOME/.local/bin"; then \
		echo "~/.local/bin is in your PATH. You're all set!"; \
	else \
		echo "Add ~/.local/bin to your PATH:"; \
		echo '  export PATH="$$HOME/.local/bin:$$PATH"'; \
	fi
else ifeq ($(UNAME),Darwin)
	@mkdir -p $(HOME)/.local/bin
	@cp $(DAEMON_BIN) $(HOME)/.local/bin/bored
	@cp $(CLI_BIN) $(HOME)/.local/bin/bore
	@cp $(TUI_BIN) $(HOME)/.local/bin/bore-tui
	@chmod +x $(HOME)/.local/bin/bored $(HOME)/.local/bin/bore $(HOME)/.local/bin/bore-tui
	@echo "Installed binaries to ~/.local/bin/"
	@# Desktop app — create .app bundle for Spotlight/Launchpad
	@if [ -f $(DESKTOP_BIN) ]; then \
		APP_DIR="$(HOME)/Applications/Bore.app"; \
		mkdir -p "$$APP_DIR/Contents/MacOS"; \
		mkdir -p "$$APP_DIR/Contents/Resources"; \
		cp $(DESKTOP_BIN) "$$APP_DIR/Contents/MacOS/bore-desktop"; \
		chmod +x "$$APP_DIR/Contents/MacOS/bore-desktop"; \
		cp desktop/Info.plist "$$APP_DIR/Contents/Info.plist"; \
		if [ -d assets/bore-desktop.iconset ] && command -v iconutil >/dev/null 2>&1; then \
			iconutil -c icns -o "$$APP_DIR/Contents/Resources/bore-desktop.icns" assets/bore-desktop.iconset; \
		elif [ -f assets/bore-desktop.icns ]; then \
			cp assets/bore-desktop.icns "$$APP_DIR/Contents/Resources/bore-desktop.icns"; \
		fi; \
		echo "Installed Bore.app to ~/Applications/ — search 'Bore' in Spotlight."; \
	fi
	@./scripts/install-service-macos.sh $(DAEMON_BIN)
	@echo ""
	@if echo "$$PATH" | tr ':' '\n' | grep -qx "$$HOME/.local/bin"; then \
		echo "~/.local/bin is in your PATH. You're all set!"; \
	else \
		echo "Add ~/.local/bin to your PATH:"; \
		echo '  export PATH="$$HOME/.local/bin:$$PATH"'; \
	fi
else
	@echo "On Windows, run: powershell -ExecutionPolicy Bypass -File scripts\\install-service-windows.ps1"
endif

uninstall:
	@echo "=== Uninstalling bore ==="
ifeq ($(UNAME),Linux)
	@./scripts/uninstall-service-linux.sh 2>/dev/null || true
	@rm -f $(HOME)/.local/bin/bored $(HOME)/.local/bin/bore $(HOME)/.local/bin/bore-tui $(HOME)/.local/bin/bore-desktop
	@rm -f $(HOME)/.local/share/applications/bore-desktop.desktop
	@for size in 16 24 32 48 64 128 256 512; do \
		rm -f "$(HOME)/.local/share/icons/hicolor/$${size}x$${size}/apps/bore.png"; \
	done
	@-update-desktop-database $(HOME)/.local/share/applications 2>/dev/null || true
	@-gtk-update-icon-cache -f -t $(HOME)/.local/share/icons/hicolor 2>/dev/null || true
	@echo "Uninstalled bore binaries and desktop entry."
	@echo "Config preserved at ~/.config/bore/"
else ifeq ($(UNAME),Darwin)
	@./scripts/uninstall-service-macos.sh 2>/dev/null || true
	@rm -f $(HOME)/.local/bin/bored $(HOME)/.local/bin/bore $(HOME)/.local/bin/bore-tui $(HOME)/.local/bin/bore-desktop
	@rm -rf "$(HOME)/Applications/Bore.app"
	@echo "Uninstalled bore binaries and desktop app."
	@echo "Config preserved at ~/Library/Application Support/bore/"
else
	@echo "On Windows, run: powershell -ExecutionPolicy Bypass -File scripts\\uninstall-service-windows.ps1"
endif
	@echo ""
	@echo "To also remove config and data, run: make purge"

proto:
	@echo "Generating protobuf code..."
	@mkdir -p $(GEN_DIR)/bore/v1
	protoc \
		--proto_path=$(PROTO_DIR) \
		--go_out=$(GEN_DIR) --go_opt=paths=source_relative \
		--go-grpc_out=$(GEN_DIR) --go-grpc_opt=paths=source_relative \
		$(PROTO_DIR)/bore/v1/*.proto

clean:
	rm -f $(DAEMON_BIN) $(CLI_BIN) $(TUI_BIN) $(DESKTOP_BIN)
	rm -f $(BIN_DIR)/bored-dev.sock
	rm -f $(GEN_DIR)/bore/v1/*.go

test: test-unit

test-unit:
	go test ./... -short -count=1

test-integration:
	go test ./... -count=1 -tags=integration

lint:
	golangci-lint run ./...

# --- Cross-platform builds ---

PLATFORMS := linux/amd64 linux/arm64 darwin/amd64 darwin/arm64 windows/amd64 windows/arm64

release: build-frontend
	@mkdir -p $(BIN_DIR)/release
	@for platform in $(PLATFORMS); do \
		GOOS=$${platform%/*} GOARCH=$${platform#*/} ; \
		ext="" ; \
		if [ "$$GOOS" = "windows" ]; then ext=".exe"; fi ; \
		echo "Building $$GOOS/$$GOARCH..." ; \
		GOOS=$$GOOS GOARCH=$$GOARCH go build $(LDFLAGS) -o $(BIN_DIR)/release/bore-$$GOOS-$$GOARCH$$ext ./cmd/bore ; \
		GOOS=$$GOOS GOARCH=$$GOARCH go build $(LDFLAGS) -o $(BIN_DIR)/release/bored-$$GOOS-$$GOARCH$$ext ./cmd/bored ; \
		GOOS=$$GOOS GOARCH=$$GOARCH go build $(LDFLAGS) -o $(BIN_DIR)/release/bore-tui-$$GOOS-$$GOARCH$$ext ./cmd/bore-tui ; \
	done
	@echo "Cross-platform CLI + daemon + TUI builds complete."
	@echo "Desktop builds require platform-native toolchains (CGo + webkit/webview)."
	@ls -lh $(BIN_DIR)/release/

# --- Installers ---

deb: build
	@if ! command -v dpkg-deb >/dev/null 2>&1; then \
		echo "Error: dpkg-deb not found. Install dpkg or run on Debian/Ubuntu."; \
		exit 1; \
	fi
	./scripts/build-deb.sh $(VERSION)

dmg: build
	@if [ "$(UNAME)" != "Darwin" ]; then \
		echo "Error: DMG can only be built on macOS."; exit 1; \
	fi
	./scripts/build-dmg.sh $(VERSION)

windows-installer: build
	@if ! command -v iscc >/dev/null 2>&1; then \
		echo "Error: Inno Setup compiler (iscc) not found."; \
		echo "Install from: https://jrsoftware.org/isinfo.php"; \
		echo "Or on CI, use: choco install innosetup"; \
		exit 1; \
	fi
	iscc /DVERSION=$(VERSION) scripts/bore-installer.iss

uninstall-macos:
	@./scripts/uninstall-macos.sh

uninstall-windows:
	@powershell -ExecutionPolicy Bypass -File scripts/uninstall-windows.ps1

release-uninstall: scripts/uninstall-macos.sh scripts/uninstall-windows.ps1
	@mkdir -p $(BIN_DIR)/release
	@cp scripts/uninstall-macos.sh $(BIN_DIR)/release/uninstall-macos.sh
	@cp scripts/uninstall-windows.ps1 $(BIN_DIR)/release/uninstall-windows.ps1
	@chmod +x $(BIN_DIR)/release/uninstall-macos.sh
	@echo "Uninstall scripts copied to $(BIN_DIR)/release/"

# --- Developer workflow ---

check:
	@echo "Checking development dependencies..."
	@echo ""
	@MISSING=0; \
	printf "  %-35s" "go (1.25+)"; \
	if command -v go >/dev/null 2>&1; then \
		GO_VER=$$(go version | sed -n 's/.*go1\.\([0-9]*\).*/\1/p'); \
		if [ -n "$$GO_VER" ] && [ "$$GO_VER" -ge 25 ] 2>/dev/null; then \
			echo "[ok] $$(go version | sed 's/go version //')"; \
		else \
			echo "[WRONG VERSION] need 1.25+, got $$(go version)"; MISSING=1; \
		fi; \
	else \
		echo "[MISSING] https://go.dev/dl/"; MISSING=1; \
	fi; \
	printf "  %-35s" "protoc"; \
	if command -v protoc >/dev/null 2>&1; then \
		echo "[ok]"; \
	else \
		echo "[MISSING] apt install protobuf-compiler / brew install protobuf"; MISSING=1; \
	fi; \
	printf "  %-35s" "protoc-gen-go"; \
	if command -v protoc-gen-go >/dev/null 2>&1; then \
		echo "[ok]"; \
	else \
		echo "[MISSING] go install google.golang.org/protobuf/cmd/protoc-gen-go@latest"; MISSING=1; \
	fi; \
	printf "  %-35s" "protoc-gen-go-grpc"; \
	if command -v protoc-gen-go-grpc >/dev/null 2>&1; then \
		echo "[ok]"; \
	else \
		echo "[MISSING] go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest"; MISSING=1; \
	fi; \
	echo ""; \
	echo "  Desktop (optional):"; \
	printf "  %-35s" "node"; \
	if command -v node >/dev/null 2>&1; then \
		echo "[ok] $$(node --version)"; \
	else \
		echo "[MISSING] https://nodejs.org/"; MISSING=1; \
	fi; \
	printf "  %-35s" "npm"; \
	if command -v npm >/dev/null 2>&1; then \
		echo "[ok] $$(npm --version)"; \
	else \
		echo "[MISSING] (installed with Node.js)"; MISSING=1; \
	fi; \
	if [ "$$(uname)" = "Linux" ]; then \
		printf "  %-35s" "pkg-config"; \
		if command -v pkg-config >/dev/null 2>&1; then \
			echo "[ok]"; \
		else \
			echo "[MISSING] apt install pkg-config"; MISSING=1; \
		fi; \
		printf "  %-35s" "webkit2gtk-4.1"; \
		if pkg-config --exists webkit2gtk-4.1 2>/dev/null; then \
			echo "[ok]"; \
		else \
			echo "[MISSING] apt install libwebkit2gtk-4.1-dev"; MISSING=1; \
		fi; \
	fi; \
	echo ""; \
	echo "  Linting (optional):"; \
	printf "  %-35s" "golangci-lint"; \
	if command -v golangci-lint >/dev/null 2>&1; then \
		echo "[ok] $$(golangci-lint version 2>&1 | head -1)"; \
	else \
		echo "[MISSING] https://golangci-lint.run/welcome/install/"; MISSING=1; \
	fi; \
	echo ""; \
	if [ $$MISSING -eq 0 ]; then \
		echo "All dependencies satisfied."; \
	else \
		echo "Some dependencies are missing (see above)."; exit 1; \
	fi

dev: proto build-daemon build-cli
	@echo ""
	@echo "Built: $(DAEMON_BIN) $(CLI_BIN)"
	@echo "Starting daemon (foreground, debug logging)..."
	@echo ""
	@echo "In another terminal, interact with:"
	@echo "  BORE_SOCKET=$(CURDIR)/bin/bored-dev.sock ./bin/bore status"
	@echo "  make dev-desktop    (desktop GUI)"
	@echo ""
	@echo "Press Ctrl+C to stop."
	@echo ""
	@BORE_LOG_LEVEL=debug BORE_SOCKET=$(CURDIR)/bin/bored-dev.sock $(DAEMON_BIN)

DEV_SOCKET := $(CURDIR)/bin/bored-dev.sock

dev-desktop: proto build-desktop
	@echo ""
	@echo "Built: $(DESKTOP_BIN)"
	@echo "Connecting to dev daemon at $(DEV_SOCKET)"
	@echo ""
	@if [ ! -S "$(DEV_SOCKET)" ]; then \
		echo "Error: Dev daemon not running. Start it first with: make dev"; \
		exit 1; \
	fi
	@BORE_SOCKET=$(DEV_SOCKET) $(DESKTOP_BIN)

purge: clean
	@echo "=== Purging all bore artifacts ==="
ifeq ($(UNAME),Linux)
	@./scripts/uninstall-service-linux.sh 2>/dev/null || true
	@rm -f $(HOME)/.local/bin/bored $(HOME)/.local/bin/bore $(HOME)/.local/bin/bore-tui $(HOME)/.local/bin/bore-desktop
	@rm -f $(HOME)/.local/share/applications/bore-desktop.desktop
	@for size in 16 24 32 48 64 128 256 512; do \
		rm -f "$(HOME)/.local/share/icons/hicolor/$${size}x$${size}/apps/bore.png"; \
	done
	@-update-desktop-database $(HOME)/.local/share/applications 2>/dev/null || true
	@-gtk-update-icon-cache -f -t $(HOME)/.local/share/icons/hicolor 2>/dev/null || true
	@rm -rf $(HOME)/.config/bore
	@rm -rf $(HOME)/.local/share/bore
	@rm -f $(HOME)/.config/systemd/user/bored.service
	@echo ""
	@echo "Removed:"
	@echo "  ./bin/                        (build artifacts)"
	@echo "  gen/bore/v1/*.go              (generated proto)"
	@echo "  ~/.local/bin/bore*            (installed binaries)"
	@echo "  ~/.config/bore/               (configuration)"
	@echo "  ~/.local/share/bore/          (database, socket, lock)"
	@echo "  bored.service                 (systemd user service)"
else ifeq ($(UNAME),Darwin)
	@./scripts/uninstall-service-macos.sh 2>/dev/null || true
	@rm -f $(HOME)/.local/bin/bored $(HOME)/.local/bin/bore $(HOME)/.local/bin/bore-tui $(HOME)/.local/bin/bore-desktop
	@rm -rf "$(HOME)/Applications/Bore.app"
	@rm -rf "$(HOME)/Library/Application Support/bore"
	@rm -rf "$(HOME)/Library/Logs/bore"
	@echo ""
	@echo "Removed:"
	@echo "  ./bin/                                      (build artifacts)"
	@echo "  gen/bore/v1/*.go                            (generated proto)"
	@echo "  ~/.local/bin/bore*                          (installed binaries)"
	@echo "  ~/Applications/Bore.app                     (desktop app)"
	@echo "  ~/Library/Application Support/bore/         (configuration & data)"
	@echo "  ~/Library/Logs/bore/                        (logs)"
	@echo "  com.bore.daemon.plist                       (launchd agent)"
else
	@echo "On Windows, run: powershell -ExecutionPolicy Bypass -File scripts\\uninstall-service-windows.ps1"
	@echo "Then manually remove %%APPDATA%%\\Bore and %%LOCALAPPDATA%%\\Bore"
endif
	@echo ""
	@echo "bore is fully removed from this system."
