package config

import (
	"os"
	"path/filepath"
)

// ConfigPath returns the path to tunnels.yaml.
func ConfigPath() string {
	return filepath.Join(ConfigDir(), "tunnels.yaml")
}

// DatabasePath returns the path to the SQLite database.
func DatabasePath() string {
	return filepath.Join(DataDir(), "state.db")
}

// LockPath returns the path to the daemon lock file.
func LockPath() string {
	return filepath.Join(DataDir(), "bored.lock")
}

// EnsureDirs creates config and data directories if they don't exist.
func EnsureDirs() error {
	if err := os.MkdirAll(ConfigDir(), 0o700); err != nil {
		return err
	}
	return os.MkdirAll(DataDir(), 0o700)
}
