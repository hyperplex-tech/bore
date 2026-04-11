# Developer Guide

## Supported Platforms

bore runs on Linux, macOS, and Windows across both x86_64 and ARM64 architectures.

| Component | Linux | macOS | Windows |
|-----------|-------|-------|---------|
| CLI (`bore`) | amd64, arm64 | amd64, arm64 | amd64, arm64 |
| Daemon (`bored`) | amd64, arm64 | amd64, arm64 | amd64, arm64 |
| TUI (`bore-tui`) | amd64, arm64 | amd64, arm64 | amd64, arm64 |
| Desktop (`bore-desktop`) | amd64 | amd64, arm64 | amd64, arm64 |

The desktop app requires CGo and platform-native webview libraries (webkit2gtk on Linux, WebKit on macOS, WebView2 on Windows). CLI, daemon, and TUI are pure Go and cross-compile to all architectures without CGo.

> **Note:** The Linux ARM64 desktop build is not currently supported because cross-compiling webkit2gtk requires a native ARM toolchain. If you have an ARM64 Linux machine, you can build it natively with `make build-desktop`.

## Prerequisites

**Required** (CLI, daemon, TUI):

- Go 1.25+
- protoc (protobuf compiler)
- protoc-gen-go
- protoc-gen-go-grpc

**Desktop app only:**

- Node.js + npm
- Linux: pkg-config, libwebkit2gtk-4.1-dev, gcc/build-essential
- macOS: Xcode command line tools
- Windows: WebView2 runtime (bundled with Windows 11, auto-downloaded by Wails on Windows 10)

**Linting:**

- golangci-lint

### Automated setup (Linux/macOS)

```bash
./scripts/dev-setup.sh          # interactive
./scripts/dev-setup.sh --auto   # non-interactive (CI)
```

### Manual verification

```bash
make check
```

Prints a status table of all dependencies with install instructions for anything missing.

## Makefile reference

All targets are listed below, grouped by purpose. Run `make <target>` from the project root.

### Quick reference

| Target | Description |
|--------|-------------|
| `all` | Generate protobuf code, then build all binaries |
| `build` | Build all binaries (CLI, daemon, TUI, desktop) |
| `build-daemon` | Build just the daemon (`bin/bored`) |
| `build-cli` | Build just the CLI (`bin/bore`) |
| `build-tui` | Build just the TUI (`bin/bore-tui`) |
| `build-desktop` | Build the desktop app (`bin/bore-desktop`); builds frontend first |
| `build-frontend` | Install npm deps and build the React frontend |
| `proto` | Regenerate protobuf Go code from `.proto` definitions |
| `test` | Run unit tests (alias for `test-unit`) |
| `test-unit` | Run unit tests in short mode |
| `test-integration` | Run full integration tests |
| `lint` | Run golangci-lint |
| `check` | Print dependency status table with install hints |
| `dev` | Build daemon + CLI, run daemon in foreground with debug logging |
| `dev-desktop` | Build and launch the desktop app connected to the dev daemon |
| `install` | Build and install all binaries + desktop entry + daemon service |
| `uninstall` | Remove installed binaries + daemon service; keep config |
| `purge` | Remove everything: binaries, service, config, data, build artifacts |
| `clean` | Remove build artifacts (`./bin/`) and generated proto code |
| `release` | Cross-compile CLI + daemon + TUI for all release platforms |
| `deb` | Build a `.deb` package for Debian/Ubuntu |
| `dmg` | Build a macOS `.dmg` installer (macOS only) |
| `windows-installer` | Build a Windows `.exe` installer via Inno Setup |

### Build targets

#### `all`

```bash
make all    # or just: make
```

Runs `proto` then `build`. This is the default target.

#### `build`

```bash
make build
```

Builds all four binaries into `./bin/`:

| Binary | Source | Description |
|--------|--------|-------------|
| `bin/bore` | `cmd/bore/` | CLI client |
| `bin/bored` | `cmd/bored/` | Background daemon |
| `bin/bore-tui` | `cmd/bore-tui/` | Terminal UI |
| `bin/bore-desktop` | `cmd/bore-desktop/` | Desktop GUI (Wails) |

All binaries are compiled with `-s -w` (stripped) and the version stamped via `-X` ldflags from `git describe`.

#### `build-daemon`, `build-cli`, `build-tui`

```bash
make build-daemon   # just bin/bored
make build-cli      # just bin/bore
make build-tui      # just bin/bore-tui
```

Build individual binaries. Useful when iterating on a single component.

#### `build-desktop`

```bash
make build-desktop
```

Depends on `build-frontend`. Builds the Wails desktop binary with the `production` build tag. On Linux, the `webkit2_41` tag is added automatically.

#### `build-frontend`

```bash
make build-frontend
```

Runs `npm install && npm run build` inside `desktop/frontend/`. Produces the bundled React app in `desktop/frontend/dist/`, which is embedded into the Go binary via `//go:embed`.

#### `proto`

```bash
make proto
```

Regenerates Go code from protobuf definitions:

- **Input**: `api/proto/bore/v1/*.proto`
- **Output**: `gen/bore/v1/*.go`
- **Tools used**: `protoc`, `protoc-gen-go`, `protoc-gen-go-grpc`

Run this after modifying any `.proto` file.

### Testing & linting

#### `test` / `test-unit`

```bash
make test         # alias for test-unit
make test-unit
```

Runs `go test ./... -short -count=1`. The `-short` flag skips long-running tests; `-count=1` disables test caching.

#### `test-integration`

```bash
make test-integration
```

Runs `go test ./... -count=1 -tags=integration`. Integration tests may start real daemons, connect to SSH servers, or use the database.

#### `lint`

```bash
make lint
```

Runs `golangci-lint run ./...`. Requires golangci-lint to be installed.

### Developer workflow

#### `check`

```bash
make check
```

Prints a status table of all required and optional dependencies. For each one, shows `[ok]`, `[MISSING]` with install instructions, or `[WRONG VERSION]`. Exits with code 1 if anything required is missing.

Checks: Go 1.25+, protoc, protoc-gen-go, protoc-gen-go-grpc, Node.js, npm, pkg-config (Linux), webkit2gtk-4.1 (Linux), golangci-lint.

#### `dev`

```bash
make dev
```

The primary development target. Runs `proto`, then `build-daemon` and `build-cli`, then starts the daemon in the foreground with:

- `BORE_LOG_LEVEL=debug` — verbose logging
- `BORE_SOCKET=./bin/bored-dev.sock` — dev-specific IPC socket, so it never conflicts with a production install

In a second terminal, interact with the dev daemon:

```bash
BORE_SOCKET=$(pwd)/bin/bored-dev.sock ./bin/bore status
```

#### `dev-desktop`

```bash
make dev-desktop
```

Runs `proto`, then `build-desktop`, then launches the desktop app pointed at the dev daemon socket. The dev daemon **must already be running** (`make dev` in another terminal). Errors immediately if the dev socket doesn't exist.

### Install / uninstall (from source)

These targets are for installing from a local build. For distributable installers, see the [Installer targets](#installer-targets) section.

#### `install`

```bash
make install
```

Depends on `build`. Installs all binaries, the desktop app entry, and the background daemon service. Behavior is platform-specific:

**Linux:**

| What | Where |
|------|-------|
| `bore`, `bored`, `bore-tui` | `~/.local/bin/` |
| `bore-desktop` | `~/.local/bin/` |
| `.desktop` file | `~/.local/share/applications/bore-desktop.desktop` |
| Icons (16-512px PNGs) | `~/.local/share/icons/hicolor/<size>/apps/bore.png` |
| systemd user service | `~/.config/systemd/user/bored.service` |
| Config directory | `~/.config/bore/` |
| Data directory | `~/.local/share/bore/` |

After install, the daemon is enabled and started via `systemctl --user enable --now bored`. If systemd is not available, prints instructions to start manually. Refreshes the desktop database and icon cache so the app appears in launchers immediately. Checks if `~/.local/bin` is in `PATH` and warns if not.

**macOS:**

| What | Where |
|------|-------|
| `bore`, `bored`, `bore-tui` | `~/.local/bin/` |
| `Bore.app` bundle | `~/Applications/Bore.app` |
| `.icns` icon | Inside `Bore.app/Contents/Resources/` (generated from `assets/bore-desktop.iconset/` via `iconutil`) |
| launchd agent | `~/Library/LaunchAgents/com.bore.daemon.plist` |

The `.app` bundle is searchable via Spotlight and Launchpad. The launchd agent auto-starts the daemon on login.

**Windows:**

Prints instructions to run the PowerShell install script:

```powershell
powershell -ExecutionPolicy Bypass -File scripts\install-service-windows.ps1
```

The script installs all binaries to `%LOCALAPPDATA%\Bore\`, creates a Start Menu shortcut, and registers a scheduled task for the daemon.

#### `uninstall`

```bash
make uninstall
```

Removes installed binaries, desktop entries, icons, and the daemon service. **Configuration and data are preserved.**

| Platform | What it removes |
|----------|----------------|
| Linux | `~/.local/bin/bore*`, `.desktop` file, hicolor icons, systemd service |
| macOS | `~/.local/bin/bore*`, `~/Applications/Bore.app`, launchd agent |
| Windows | Prints instructions to run the PowerShell uninstall script |

#### `purge`

```bash
make purge
```

Runs `clean` first, then removes **everything** — binaries, service, configuration, and data.

| Platform | What it removes (in addition to `uninstall`) |
|----------|----------------------------------------------|
| Linux | `~/.config/bore/`, `~/.local/share/bore/`, `~/.config/systemd/user/bored.service` |
| macOS | `~/Library/Application Support/bore/`, `~/Library/Logs/bore/` |
| Windows | Prints instructions; suggests manual removal of `%APPDATA%\Bore` and `%LOCALAPPDATA%\Bore` |

#### `clean`

```bash
make clean
```

Removes local build artifacts only (does not touch installed files):

- `bin/bore`, `bin/bored`, `bin/bore-tui`, `bin/bore-desktop`
- `bin/bored-dev.sock`
- `gen/bore/v1/*.go`

### Installer targets

These produce distributable packages for end users. Each depends on `build`.

#### `deb`

```bash
make deb
```

Builds a `.deb` package at `bin/bore_<version>_<arch>.deb` using `dpkg-deb`.

**Requires:** `dpkg-deb` (installed by default on Debian/Ubuntu).

**Package contents:**

| File | Path in package |
|------|----------------|
| CLI, daemon, TUI, desktop | `/usr/bin/` |
| `.desktop` entry | `/usr/share/applications/bore-desktop.desktop` |
| Icons (16-512px) | `/usr/share/icons/hicolor/<size>/apps/bore.png` |
| systemd user service | `/usr/lib/systemd/user/bored.service` |
| Copyright | `/usr/share/doc/bore/copyright` |

Post-install script refreshes the desktop database and icon cache, and reminds the user to enable the daemon:

```bash
sudo dpkg -i bin/bore_*.deb
systemctl --user enable --now bored
```

Uninstall:

```bash
systemctl --user disable --now bored
sudo apt remove bore        # keep config
sudo apt purge bore         # remove everything
```

**Note:** The daemon runs as a systemd **user** service (not system-wide). Use `systemctl --user status bored` to check it, not `sudo systemctl status bored`.

#### `dmg`

```bash
make dmg
```

Builds a macOS `.dmg` installer at `bin/Bore-<version>.dmg` using `hdiutil`.

**Requires:** macOS (uses `hdiutil` and `iconutil`, both built-in).

**DMG contents:**

| Item | Purpose |
|------|---------|
| `Bore.app` | Desktop GUI with all binaries bundled in `Contents/MacOS/` |
| `Applications` symlink | Drag `Bore.app` here to install |
| `Install CLI Tools.command` | Double-click to copy CLI tools to `~/.local/bin/` and register the launchd daemon |

The `.app` bundle includes `bore`, `bored`, and `bore-tui` alongside the desktop binary so the "Install CLI Tools" script can copy them out.

The version in `Info.plist` is stamped from the git tag. The `.icns` icon is generated from `assets/bore-desktop.iconset/` via `iconutil`.

> **Important:** The app is not currently code-signed. After dragging `Bore.app` to `/Applications`, users must clear the macOS quarantine flag before launching:
> ```bash
> xattr -cr /Applications/Bore.app
> ```

#### `windows-installer`

```bash
make windows-installer
```

Builds a Windows installer at `bin/Bore-Setup-<version>.exe` using Inno Setup.

**Requires:** Inno Setup 6+ (`iscc` on PATH). Install via `choco install innosetup` on CI.

**What the installer does:**

- Modern wizard UI with the bore icon
- Installs all four binaries to `%LOCALAPPDATA%\Bore\`
- Creates Start Menu shortcuts (Bore + Uninstall)
- Optional: desktop shortcut
- Optional: add install directory to user `PATH`
- Registers `bored.exe` as a scheduled task (`/sc onlogon /rl limited`)
- Post-install: offers to start the daemon and launch the app
- Uninstaller accessible from Start Menu and "Add or Remove Programs"

**What the uninstaller does:**

- Stops the daemon scheduled task
- Removes the scheduled task
- Deletes binaries, shortcuts, and runtime files (`bore.db`, `bored.lock`)
- Config at `%APPDATA%\Bore\` is preserved

### Release builds

#### `release`

```bash
make release
```

Cross-compiles CLI, daemon, and TUI for all supported platforms. Runs `build-frontend` first.

| Platform | Architectures |
|----------|--------------|
| linux | amd64, arm64 |
| darwin | amd64, arm64 |
| windows | amd64, arm64 |

Output goes to `bin/release/` with filenames like `bore-linux-amd64`, `bored-darwin-arm64`, `bore-tui-windows-arm64.exe`.

Desktop builds are **not** included in cross-compilation because they require CGo and platform-native webview libraries. Build those natively on each platform with `make build-desktop`.

## Platform-specific paths

| | Linux | macOS | Windows |
|---|---|---|---|
| Config | `~/.config/bore/` | `~/Library/Application Support/bore/` | `%APPDATA%\Bore\` |
| Data | `~/.local/share/bore/` | `~/Library/Application Support/bore/` | `%LOCALAPPDATA%\Bore\` |
| IPC | Unix socket | Unix socket | Named pipe |

XDG environment variables (`XDG_CONFIG_HOME`, `XDG_DATA_HOME`) are respected on Linux and macOS.

## Backup and restore

```bash
./scripts/backup-config.sh ~/backups
./scripts/restore-config.sh ~/backups/bore-backup-20260409_120000.tar.gz
```

Existing files are saved to `.bak` before overwriting.

## CI/CD

GitHub Actions workflows live in `.github/workflows/`.

### CI (`.github/workflows/ci.yml`)

Runs on every push to `main` and every pull request. Three parallel jobs:

| Job | Runner | What it does |
|-----|--------|-------------|
| **Test** | ubuntu-latest | `make proto` + `make test` |
| **Lint** | ubuntu-latest | golangci-lint via official action |
| **Build** | ubuntu, macos, windows | Builds CLI + daemon + TUI for all 6 OS/arch combos |

The build matrix covers:

| OS | Architectures |
|----|--------------|
| linux | amd64, arm64 |
| darwin | amd64, arm64 |
| windows | amd64, arm64 |

### Release (`.github/workflows/release.yml`)

Triggered by pushing a tag matching `v*` (e.g. `v1.0.0`, `v2.0.0-beta.1`).

**How to cut a release:**

```bash
git tag v1.0.0
git push origin v1.0.0
```

The workflow runs 5 jobs:

| Job | Runner | Produces |
|-----|--------|---------|
| **build-cli** | ubuntu-latest | Cross-compiled CLI + daemon + TUI for all 6 platform/arch combos |
| **build-linux-deb** (x2) | ubuntu-latest | `.deb` packages for amd64 and arm64 |
| **build-macos-dmg** (x2) | macos-latest | `.dmg` installers for arm64 (Apple Silicon) and amd64 (Intel) |
| **build-windows-installer** (x2) | windows-latest | `.exe` installers for amd64 and arm64 |
| **release** | ubuntu-latest | Collects all artifacts, generates changelog, creates GitHub Release |

**What gets published to each release:**

| Asset | Description |
|-------|-------------|
| `bore_<ver>_amd64.deb` | Linux x86_64 installer (includes desktop app) |
| `bore_<ver>_arm64.deb` | Linux ARM64 installer (no desktop app) |
| `Bore-<ver>-arm64.dmg` | macOS Apple Silicon installer |
| `Bore-<ver>-amd64.dmg` | macOS Intel installer |
| `Bore-Setup-<ver>-amd64.exe` | Windows x86_64 installer |
| `Bore-Setup-<ver>-arm64.exe` | Windows ARM64 installer |
| `bore-linux-amd64.tar.gz` | Standalone CLI binary |
| `bored-linux-amd64.tar.gz` | Standalone daemon binary |
| `bore-tui-linux-amd64.tar.gz` | Standalone TUI binary |
| _(same pattern for all 6 platform/arch combos)_ | |

Tags containing `-rc`, `-beta`, or `-alpha` are automatically marked as pre-releases.

### Code signing

Release builds are **not currently signed**. This means:

- **Windows:** Users will see a SmartScreen "Windows protected your PC" warning when running the installer. They must click "More info" → "Run anyway". Windows Defender may also flag Go binaries as false positives.
- **macOS:** Users will see a Gatekeeper warning and must run `xattr -cr /Applications/Bore.app` or right-click → Open to bypass quarantine.

To eliminate these warnings, the following are needed:

| Platform | Requirement | Cost | Effect |
|----------|------------|------|--------|
| **Windows** | EV Code Signing Certificate (e.g. SSL.com, DigiCert, Sectigo) | ~$300-400/yr | Immediate SmartScreen trust, no Defender false positives |
| **macOS** | Apple Developer Program + notarization | $99/yr | No Gatekeeper warning, no `xattr` needed |

The `build-dmg.sh` script supports signing and notarization via environment variables (`CODESIGN_IDENTITY`, `NOTARIZE_APPLE_ID`, `NOTARIZE_TEAM_ID`, `NOTARIZE_PASSWORD`) but these are not currently configured in CI.

**Architecture notes:**

- CLI, daemon, and TUI are pure Go — cross-compiled from a single Ubuntu runner for all 6 targets.
- Desktop app requires CGo + native webview — built natively on each platform's runner.
- Linux ARM64 `.deb` does not include the desktop app (cross-compiling webkit2gtk requires a native ARM toolchain).
- macOS builds use the `macos-latest` (Apple Silicon) runner with `GOARCH=amd64` for Intel cross-compilation, which works because macOS provides universal SDK support.
- Windows ARM64 desktop app cross-compiles from an amd64 runner because Wails uses WebView2 via COM (no CGo required on Windows).

## Project structure

```
bore/
  .github/
    workflows/
      ci.yml                      CI: test, lint, build on push/PR
      release.yml                 Release: build installers, publish to GitHub Releases
  assets/
    icon.png                    source icon (1254x1254)
    icon.ico                    Windows icon (multi-size)
    icon-NxN.png                Linux hicolor icons (16-512px)
    bore-desktop.iconset/       macOS iconset (for iconutil)
  cmd/
    bore/                       CLI entry point
    bored/                      daemon entry point
    bore-tui/                   TUI entry point
    bore-desktop/               desktop GUI entry point
  internal/
    config/                     paths (platform-specific), YAML config, watcher
    ipc/                        IPC transport (Unix socket / Windows named pipe)
    auth/                       SSH authentication (agent, keys, certs)
    daemon/                     daemon lifecycle, locking, signals (platform-specific)
    engine/                     tunnel engine (SSH, SOCKS, K8s)
    ...
  api/proto/                    protobuf service definitions
  gen/                          generated protobuf Go code
  desktop/
    frontend/                   React + Vite + TypeScript + Tailwind
    app.go                      Wails backend bindings
    bore-desktop.desktop        Linux .desktop entry
    Info.plist                  macOS .app bundle metadata
  scripts/
    dev-setup.sh                interactive dependency installer
    install-service-linux.sh    systemd user service setup
    install-service-macos.sh    launchd user agent setup
    install-service-windows.ps1 Task Scheduler + Start Menu + CLI setup
    uninstall-service-linux.sh  systemd service removal
    uninstall-service-macos.sh  launchd agent removal
    uninstall-service-windows.ps1 Task Scheduler + shortcut + binary removal
    build-deb.sh                .deb package builder
    build-dmg.sh                macOS .dmg builder
    bore-installer.iss          Windows Inno Setup script
    generate-icons.py           regenerate platform icons from icon.png
    backup-config.sh            config backup
    restore-config.sh           config restore
```
