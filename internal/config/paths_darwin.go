//go:build darwin

package config

import (
	"os"
	"path/filepath"
)

// ConfigDir returns the bore config directory.
// Uses ~/Library/Application Support/bore/ by default, or XDG_CONFIG_HOME if set.
func ConfigDir() string {
	if xdg := os.Getenv("XDG_CONFIG_HOME"); xdg != "" {
		return filepath.Join(xdg, "bore")
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, "Library", "Application Support", "bore")
}

// DataDir returns the bore data directory.
// Uses ~/Library/Application Support/bore/ by default, or XDG_DATA_HOME if set.
func DataDir() string {
	if xdg := os.Getenv("XDG_DATA_HOME"); xdg != "" {
		return filepath.Join(xdg, "bore")
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, "Library", "Application Support", "bore")
}

// SocketPath returns the path to the daemon Unix socket.
func SocketPath() string {
	return filepath.Join(DataDir(), "bored.sock")
}
