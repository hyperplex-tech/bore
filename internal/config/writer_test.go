package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestAddTunnelNewGroup(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "tunnels.yaml")

	tc := TunnelConfig{
		Name:       "test-tunnel",
		LocalPort:  5432,
		RemoteHost: "db.internal",
		RemotePort: 5432,
		SSHHost:    "bastion.example.com",
	}

	if err := AddTunnel(path, "test-group", tc); err != nil {
		t.Fatalf("AddTunnel: %v", err)
	}

	// Verify file was written.
	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	g, ok := cfg.Groups["test-group"]
	if !ok {
		t.Fatal("group test-group not found")
	}
	if len(g.Tunnels) != 1 {
		t.Fatalf("expected 1 tunnel, got %d", len(g.Tunnels))
	}
	if g.Tunnels[0].Name != "test-tunnel" {
		t.Errorf("expected name test-tunnel, got %s", g.Tunnels[0].Name)
	}
}

func TestAddTunnelDuplicate(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "tunnels.yaml")

	tc := TunnelConfig{Name: "dup", LocalPort: 1234, RemoteHost: "x", RemotePort: 1}
	if err := AddTunnel(path, "g", tc); err != nil {
		t.Fatal(err)
	}
	if err := AddTunnel(path, "g", tc); err == nil {
		t.Fatal("expected duplicate error")
	}
}

func TestAddTunnelExistingGroup(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "tunnels.yaml")

	// Create initial config.
	cfg := Defaults()
	cfg.Groups["dev"] = Group{
		Description: "Dev",
		Tunnels: []TunnelConfig{
			{Name: "existing", LocalPort: 1000, RemoteHost: "a", RemotePort: 1},
		},
	}
	if err := Save(path, &cfg); err != nil {
		t.Fatal(err)
	}

	// Add a second tunnel to the same group.
	tc := TunnelConfig{Name: "new-tunnel", LocalPort: 2000, RemoteHost: "b", RemotePort: 2}
	if err := AddTunnel(path, "dev", tc); err != nil {
		t.Fatal(err)
	}

	cfg2, _ := Load(path)
	g := cfg2.Groups["dev"]
	if len(g.Tunnels) != 2 {
		t.Fatalf("expected 2 tunnels, got %d", len(g.Tunnels))
	}
}

func TestSaveAtomicity(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")

	cfg := Defaults()
	if err := Save(path, &cfg); err != nil {
		t.Fatal(err)
	}

	// Verify no temp file left behind.
	if _, err := os.Stat(path + ".tmp"); !os.IsNotExist(err) {
		t.Fatal("temp file should not exist after save")
	}

	// Verify the file is valid YAML.
	if _, err := Load(path); err != nil {
		t.Fatalf("saved config not loadable: %v", err)
	}
}
