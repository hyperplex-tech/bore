//go:build linux

package config

import (
	"os"
	"path/filepath"
)

// ConfigDir returns the bore config directory (~/.config/bore/).
func ConfigDir() string {
	if xdg := os.Getenv("XDG_CONFIG_HOME"); xdg != "" {
		return filepath.Join(xdg, "bore")
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", "bore")
}

// DataDir returns the bore data directory (~/.local/share/bore/).
func DataDir() string {
	if xdg := os.Getenv("XDG_DATA_HOME"); xdg != "" {
		return filepath.Join(xdg, "bore")
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".local", "share", "bore")
}

// SocketPath returns the path to the daemon Unix socket.
func SocketPath() string {
	return filepath.Join(DataDir(), "bored.sock")
}
