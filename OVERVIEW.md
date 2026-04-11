# Technical Overview

This document covers the architecture, internals, and design decisions behind Bore. For user-facing docs, see [README.md](README.md). For build instructions and Makefile reference, see [DEVELOPERS.md](DEVELOPERS.md).

## Architecture

Bore uses a **daemon + client** architecture. The daemon (`bored`) is the single source of truth for all tunnel state. Clients (CLI, TUI, desktop) are thin frontends that communicate with the daemon over gRPC.

```
┌─────────────────────────────────────────────────────────────────┐
│                          Clients                                │
│                                                                 │
│   bore (CLI)       bore-tui (TUI)       bore-desktop (GUI)     │
│   Cobra commands   Bubble Tea app       Wails + React app      │
│                                                                 │
└────────────────────────┬────────────────────────────────────────┘
                         │
                    gRPC over IPC
              Unix socket (Linux/macOS)
              Named pipe (Windows)
                         │
┌────────────────────────┴────────────────────────────────────────┐
│                       Daemon (bored)                            │
│                                                                 │
│  ┌─────────┐  ┌──────────┐  ┌─────────┐  ┌──────────────────┐ │
│  │ gRPC    │  │  Engine   │  │ Config  │  │  Event Bus       │ │
│  │ Server  │──│          │──│ Watcher │──│  (pub/sub)       │ │
│  │         │  │  Tunnels  │  │         │  │                  │ │
│  │ 4 svcs  │  │  Mux      │  │ YAML    │  │  State changes → │ │
│  │         │  │  Health   │  │ reload  │  │  all subscribers │ │
│  └─────────┘  │  Hooks    │  └─────────┘  └──────────────────┘ │
│               │  Auth     │                                     │
│               │  Ports    │  ┌──────────────────┐              │
│               └──────────┘  │  Store (SQLite)   │              │
│                              │  State + logs     │              │
│                              └──────────────────┘              │
└─────────────────────────────────────────────────────────────────┘
         │                    │                     │
    SSH tunnels         K8s port-fwd          SOCKS5 proxy
  (x/crypto/ssh)     (kubectl subprocess)   (RFC 1928 impl)
```

### Why a daemon?

SSH tunnels die when the process that created them exits. A daemon means:

- Tunnels survive terminal closures and laptop sleep/wake cycles
- A single process manages all connections (no duplicate SSH sessions)
- Any client can query and control tunnels without owning them
- Config reload and auto-reconnect happen in one place

The daemon uses `oklog/run` for actor-based lifecycle management: the gRPC server, config watcher, and event logger run as independent actors that start together and shut down cleanly on SIGINT/SIGTERM.

## IPC Transport

Clients and daemon communicate via gRPC over a local IPC channel:

| Platform | Transport | Default path |
|----------|-----------|-------------|
| Linux | Unix socket | `$XDG_DATA_HOME/bore/bored.sock` (typically `~/.local/share/bore/bored.sock`) |
| macOS | Unix socket | `~/Library/Application Support/bore/bored.sock` |
| Windows | Named pipe | `\\.\pipe\bore-daemon` |

Override with `BORE_SOCKET` environment variable or `--socket` flag.

Implementation: `internal/ipc/ipc_unix.go` and `internal/ipc/ipc_windows.go` with build tags.

## gRPC Services

Defined in `api/proto/bore/v1/`:

### TunnelService (`tunnel.proto`)

Core tunnel operations:

- `Connect(names[], group)` -- start tunnels by name or group
- `Disconnect(names[], group)` -- stop tunnels
- `DisconnectAll()` -- stop everything
- `List(group, status)` -- list tunnels with optional filtering
- `Pause(name)` -- graceful disconnect (stays in config)
- `Retry(name)` -- retry a failed tunnel
- `GetLogs(name, tail, follow)` -- historical + streaming log entries

### DaemonService (`daemon.proto`)

Daemon lifecycle:

- `Status()` -- version, tunnel counts, SSH agent info, Tailscale status, config path
- `Shutdown()` -- graceful daemon stop
- `ReloadConfig()` -- trigger config re-read and reconciliation

### GroupService (`group.proto`)

Group management:

- `ListGroups()` -- all groups with tunnel/active counts
- `AddGroup(name, description)`
- `RenameGroup(old, new)`
- `DeleteGroup(name)`

### EventService (`events.proto`)

Real-time state updates (server-streaming):

- `Subscribe(types[])` -- stream of events, optionally filtered by type

Event types: `TunnelConnected`, `TunnelDisconnected`, `TunnelError`, `TunnelRetrying`, `ConfigReloaded`.

The TUI and desktop app subscribe to this stream for real-time UI updates without polling.

## Tunnel Engine

The engine (`internal/engine/`) is the core of the daemon. It owns the lifecycle of every tunnel.

### Tunnel types

**Local SSH** (`internal/engine/tunnel.go`): Binds a local TCP listener, accepts connections, opens an SSH `direct-tcpip` channel for each, and runs bidirectional `io.Copy`. Each forwarded connection runs in its own goroutine.

**Remote SSH**: The SSH server binds a port and forwards traffic back to a local target. Reverse of local forwarding.

**Dynamic SOCKS5** (`internal/engine/socks.go`): Full RFC 1928 SOCKS5 implementation. Handles version negotiation, no-auth method, CONNECT command, and supports IPv4, IPv6, and domain address types. All connections route through the SSH client. Used when `type: dynamic`.

**Kubernetes** (`internal/engine/k8s.go`): Wraps `kubectl port-forward` as a managed child process. Supports `--context`, `--namespace`, `--address`. Watches stdout for the "Forwarding from" confirmation, probes the local port for readiness, captures stderr for error reporting. Same `Connect`/`Disconnect`/`Info` interface as SSH tunnels.

### Tunnel lifecycle

```
STOPPED ──→ CONNECTING ──→ ACTIVE
               │               │
               ↓               │ (connection lost)
            ERROR ←────────────┘
               │
               ↓
           RETRYING ──→ CONNECTING ──→ ACTIVE
               │
               ↓ (user pause)
            PAUSED
```

State transitions publish events to the event bus, which propagates to all subscribed clients and the SQLite logger.

### TCP forwarding (`internal/engine/forward.go`)

Bidirectional relay with proper half-close handling. When one side sends EOF, the relay calls `CloseWrite` on the other side rather than closing the full connection. This is critical for protocols like HTTP and databases that depend on clean half-close semantics.

## SSH Connection Multiplexing

`internal/engine/mux.go`

When multiple tunnels target the same SSH server (`user@host:port`), they share a single SSH connection. This reduces authentication overhead and makes group connects nearly instant after the first tunnel.

- **Key**: `user@host:port` string
- **Reference counting**: Each tunnel Acquire increments, Release decrements
- **Liveness probe on Acquire**: Sends a `keepalive@bore` request to detect stale connections before reusing them
- **Grace period**: 30 seconds after last release before closing idle connections
- **Background reaper**: Checks every 10 seconds for expired idle connections
- **Race-safe**: Double-check pattern on dial prevents duplicate connections when two tunnels race to the same endpoint

## Health Monitoring

`internal/health/monitor.go`

Each active tunnel gets a health watcher that sends periodic SSH keepalive requests:

- **Interval**: Configurable via `defaults.keepalive_interval` (default 30s)
- **Mechanism**: SSH `keepalive@bore` global request
- **States**: Healthy (all keepalives succeed) / Degraded (some failures) / Dead (max consecutive failures reached)
- **On Dead**: Triggers tunnel teardown + reconnect loop

### Reconnect with backoff (`internal/health/backoff.go`)

Exponential backoff with jitter: 1s, 2s, 4s, 8s, 16s, 32s, 60s (capped). The cap is configurable via `defaults.reconnect_max_interval`. A +/-25% random jitter prevents thundering herd when many tunnels reconnect simultaneously.

The reconnect loop:
1. Teardown: close listener, release mux reference, release port
2. Set status to `retrying` with `next_retry_secs` countdown
3. Wait backoff delay
4. Attempt `doConnect`
5. On success: re-register health monitor, reset backoff
6. On failure: loop with increasing delay

Context cancellation (from `Disconnect` or daemon shutdown) cleanly stops the loop at any stage.

## Authentication

`internal/auth/`

Strategy pattern with four providers:

| Provider | Description | File |
|----------|-------------|------|
| `AgentProvider` | SSH agent (`SSH_AUTH_SOCK` on Unix, Pageant on Windows) | `agent.go` |
| `KeyProvider` | Private key file with auto-discovery (`~/.ssh/id_ed25519`, `id_rsa`, `id_ecdsa`) | `key.go` |
| `CertProvider` | Certificate-based auth (private key + `*-cert.pub` companion file) | `cert.go` |
| `CompositeProvider` | Tries agent first, falls back to key discovery | `provider.go` |

The default auth method (`agent`) uses the composite provider. Each tunnel can override with `auth_method: key` or `auth_method: cert` in its config.

## Jump Hosts

`internal/engine/jump.go`

Multi-hop SSH connections via `ProxyJump`-style chains:

```
laptop → bastion.example.com → internal-bastion → target
```

Implementation: `dialSSHViaJumpHosts` builds a nested chain of SSH connections. Each hop:
1. Dials `net.Conn` to the next host through the current SSH client
2. Performs `ssh.NewClientConn` handshake over that connection
3. Creates a new `ssh.Client` for the next hop

Each hop resolves auth independently via the composite provider. The final `ssh.Client` is used for tunnel forwarding.

When jump hosts are configured, the mux's `dialFn` uses the jump chain instead of direct `ssh.Dial`.

## Hook System

`internal/hook/runner.go`

Pre-connect and post-connect shell hooks:

- Executed via `sh -c` with a 30-second timeout
- **Pre-connect failure aborts the tunnel connection**
- Post-connect failure is logged but non-fatal
- 10 environment variables injected: `BORE_TUNNEL_NAME`, `BORE_GROUP`, `BORE_LOCAL_HOST`, `BORE_LOCAL_PORT`, `BORE_REMOTE_HOST`, `BORE_REMOTE_PORT`, `BORE_SSH_HOST`, `BORE_SSH_PORT`, `BORE_SSH_USER`, `BORE_STATUS`

## Config System

### Loading (`internal/config/config.go`)

The YAML config is loaded into a `Config` struct. Each tunnel inherits defaults from the top-level `defaults` section. The `ResolvedTunnel` type pairs a tunnel with its group name and resolved default values.

### Hot-reload (`internal/config/watcher.go`)

Uses `fsnotify` to watch the config file:
- Debounces write events by 500ms (coalesces rapid writes from editors)
- Handles vim-style save (delete + rename) by re-adding the watch
- Triggers `ReloadConfig` on the daemon which runs reconciliation

### Reconciliation (`internal/engine/engine.go`)

On config reload, the engine diffs the new config against running state:
- **New tunnels**: Registered but not connected
- **Removed tunnels**: Disconnected and removed
- **Changed tunnels**: Disconnected, replaced with new config (detects port, host, user, auth changes)

Returns `(added, removed, updated)` counts.

### Paths (`internal/config/paths_*.go`)

Platform-specific path resolution with build tags:

| | Linux | macOS | Windows |
|---|---|---|---|
| Config | `$XDG_CONFIG_HOME/bore/` | `~/Library/Application Support/bore/` | `%APPDATA%\Bore\` |
| Data | `$XDG_DATA_HOME/bore/` | `~/Library/Application Support/bore/` | `%LOCALAPPDATA%\Bore\` |
| Socket | `$XDG_DATA_HOME/bore/bored.sock` | `~/Library/Application Support/bore/bored.sock` | `\\.\pipe\bore-daemon` |

## Storage

`internal/store/sqlite.go`

SQLite database (via `modernc.org/sqlite`, pure Go, no CGo) in WAL mode. Stores:

- **Tunnel state**: Current status, error message, connected-at timestamp, retry count
- **Connection log**: Timestamped event history per tunnel (connect, disconnect, error, retry)

The `EventLogger` (`internal/store/logger.go`) subscribes to the event bus and persists every tunnel event automatically.

## Event Bus

`internal/event/bus.go`

In-process pub/sub:
- Non-blocking publish (drops events if a subscriber's buffer is full)
- Buffered subscriber channels
- Thread-safe subscribe/unsubscribe
- Used by: gRPC EventService (streaming to clients), SQLite logger, engine (internal notifications)

## Port Allocator

`internal/port/allocator.go`

Dual conflict detection:
1. **Bore-internal**: Tracks claimed ports in a map to prevent two tunnels from binding the same port
2. **OS-level**: Attempts `net.Listen` to detect ports already in use by other programs

Also provides `FindFreePort` for auto-assignment when no specific port is requested.

## Tailscale Integration

`internal/tailscale/detect.go`

Optional integration:
- Runs `tailscale status --json` to detect installation and connection state
- Reports: available, connected, IP, hostname, tailnet name
- `IsTailscaleAddr(host)` checks for `.ts.net` MagicDNS suffix or Tailscale CGNAT range (`100.64.0.0/10`)
- Shown in daemon status, CLI output, TUI status bar, and desktop status bar

## SSH Config Import

`internal/profile/importer.go`

Hand-rolled SSH config parser (no dependencies):
- Parses: `Host`, `HostName`, `User`, `Port`, `IdentityFile`, `ProxyJump`, `LocalForward`
- Skips wildcard hosts (`*`, `?`)
- Multiple `LocalForward` entries from one host produce numbered tunnel names
- `ProxyJump` chains are split into `JumpHosts` slices

## Client Implementations

### CLI (`cmd/bore/`, `internal/cli/`)

Built with Cobra. Connects to the daemon via `cli.Dial()` which creates a gRPC client over the IPC socket. Each subcommand maps directly to a gRPC call.

### TUI (`cmd/bore-tui/`, `internal/tui/`)

Built with Bubble Tea v2 + Bubbles + Lip Gloss. Elm Architecture (Model-Update-View) with async message passing. Subscribes to the daemon's event stream for real-time updates. 14 files covering: app orchestrator, sidebar, tunnel list, tunnel item rendering, log viewer, status bar, key bindings, styles, and gRPC-to-Bubble-Tea command bridge.

Key bindings: `j`/`k` navigate, `c` connect, `d` disconnect, `r` retry, `l` toggle logs, `Tab` switch focus, `/` filter, `q` quit, `?` help.

### Desktop (`cmd/bore-desktop/`, `desktop/`)

Built with Wails v2 + React 19 + Vite + TypeScript + Tailwind CSS. The Go backend (`desktop/app.go`) binds methods that the React frontend calls via Wails' generated JS stubs. The frontend is embedded into the Go binary via `//go:embed`.

Event bridge: A goroutine subscribes to the daemon's gRPC event stream and emits events to the React frontend via `runtime.EventsEmit("tunnel-event")`, giving real-time UI updates.

Polling: 3-second interval for status refresh as a fallback for any missed events.

## Build & Release Pipeline

Bore uses GitHub Actions for CI and automated releases.

### Platform and architecture matrix

All CLI, daemon, and TUI binaries are pure Go (no CGo) and cross-compile from a single runner. The desktop app requires CGo and platform-native webview libraries, so it must be built on native runners.

| Component | Build method | Architectures |
|-----------|-------------|--------------|
| CLI, daemon, TUI | Cross-compiled (pure Go) | linux/amd64, linux/arm64, darwin/amd64, darwin/arm64, windows/amd64, windows/arm64 |
| Desktop (Linux) | Native (needs webkit2gtk) | amd64 only |
| Desktop (macOS) | Native + cross (universal SDK) | arm64, amd64 |
| Desktop (Windows) | Native + cross (WebView2 via COM, no CGo) | amd64, arm64 |

### CI workflow

Runs on every push to `main` and every pull request:

1. **Test** — unit tests on Linux
2. **Lint** — golangci-lint
3. **Build** — compile CLI/daemon/TUI on all 6 OS/arch combos to catch platform-specific issues

### Release workflow

Triggered by pushing a `v*` tag. Runs 5 jobs:

1. **build-cli** — cross-compiles CLI + daemon + TUI for all 6 platform/arch targets from a single Ubuntu runner
2. **build-linux-deb** — matrix job building `.deb` packages for amd64 (with desktop) and arm64 (without desktop)
3. **build-macos-dmg** — matrix job building `.dmg` installers for arm64 and amd64 on a macOS runner
4. **build-windows-installer** — matrix job building Inno Setup `.exe` installers for amd64 and arm64 on a Windows runner
5. **release** — downloads all artifacts, packages them (`.tar.gz` for Unix, `.zip` for Windows), generates a changelog from git history, and creates a GitHub Release with all assets attached

Pre-release tags (`-rc`, `-beta`, `-alpha`) are automatically marked as pre-releases on GitHub.

### Why Linux ARM64 has no desktop app

The desktop app uses Wails, which on Linux requires `webkit2gtk` — a C library. Cross-compiling C libraries for a different architecture requires a cross-compilation toolchain (`aarch64-linux-gnu-gcc`) and ARM64 versions of all system libraries. This is complex and fragile in CI. The CLI, daemon, and TUI (all pure Go) work fine on ARM64 via standard Go cross-compilation.

If you have a native ARM64 Linux machine (Raspberry Pi 4+, AWS Graviton, etc.), you can build the desktop app locally with `make build-desktop`.

### Why macOS cross-compilation works

macOS runners (Apple Silicon) can cross-compile for Intel (`GOARCH=amd64`) because Apple's SDK supports both architectures natively. The Wails/WebKit integration uses system frameworks that are available for both architectures in the same SDK.

### Why Windows ARM64 cross-compilation works

Wails on Windows uses WebView2 through COM interfaces (`go-webview2`), which does not require CGo. This means the entire Windows build — including the desktop app — is pure Go and cross-compiles freely between amd64 and arm64.

## Dependencies

| Dependency | Purpose |
|-----------|---------|
| `golang.org/x/crypto/ssh` | SSH client, tunneling, agent protocol |
| `google.golang.org/grpc` | gRPC framework for IPC |
| `google.golang.org/protobuf` | Protocol buffer serialization |
| `gopkg.in/yaml.v3` | YAML config parsing |
| `modernc.org/sqlite` | Pure-Go SQLite driver (no CGo) |
| `github.com/spf13/cobra` | CLI framework |
| `github.com/rs/zerolog` | Structured JSON logging |
| `github.com/oklog/run` | Actor-based process lifecycle |
| `github.com/fsnotify/fsnotify` | File system event watching |
| `charm.land/bubbletea/v2` | TUI framework (Elm Architecture) |
| `charm.land/bubbles/v2` | TUI components |
| `charm.land/lipgloss/v2` | TUI styling |
| `github.com/wailsapp/wails/v2` | Desktop app framework (Go + webview) |
