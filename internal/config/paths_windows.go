//go:build windows

package config

import (
	"os"
	"path/filepath"
)

// ConfigDir returns the bore config directory (%APPDATA%\Bore\).
func ConfigDir() string {
	if appData := os.Getenv("APPDATA"); appData != "" {
		return filepath.Join(appData, "Bore")
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, "AppData", "Roaming", "Bore")
}

// DataDir returns the bore data directory (%LOCALAPPDATA%\Bore\).
func DataDir() string {
	if localAppData := os.Getenv("LOCALAPPDATA"); localAppData != "" {
		return filepath.Join(localAppData, "Bore")
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, "AppData", "Local", "Bore")
}

// SocketPath returns the named pipe address for the daemon.
func SocketPath() string {
	return `\\.\pipe\bore-daemon`
}
