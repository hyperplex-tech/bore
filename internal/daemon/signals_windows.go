//go:build windows

package daemon

import "github.com/oklog/run"

// addPlatformActors is a no-op on Windows.
// Config reload is handled via the gRPC ReloadConfig RPC and the fsnotify watcher.
func addPlatformActors(g *run.Group, d *Daemon) {}
