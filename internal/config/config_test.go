package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDefaults(t *testing.T) {
	cfg := Defaults()
	if cfg.Version != 1 {
		t.Fatalf("expected version 1, got %d", cfg.Version)
	}
	if cfg.Defaults.SSHPort != 22 {
		t.Fatalf("expected default SSH port 22, got %d", cfg.Defaults.SSHPort)
	}
	if cfg.Defaults.AuthMethod != "agent" {
		t.Fatalf("expected default auth 'agent', got %q", cfg.Defaults.AuthMethod)
	}
	if !cfg.Defaults.Reconnect {
		t.Fatal("expected reconnect default true")
	}
}

func TestLoadAndApplyDefaults(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")

	yaml := `version: 1
defaults:
  ssh_port: 2222
  ssh_user: testuser
  auth_method: key
groups:
  dev:
    description: Dev
    tunnels:
      - name: db
        local_port: 5432
        remote_host: db.internal
        remote_port: 5432
        ssh_host: bastion.example.com
`
	os.WriteFile(path, []byte(yaml), 0o644)

	cfg, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}

	tunnel := cfg.Groups["dev"].Tunnels[0]
	if tunnel.SSHPort != 2222 {
		t.Fatalf("expected inherited SSH port 2222, got %d", tunnel.SSHPort)
	}
	if tunnel.SSHUser != "testuser" {
		t.Fatalf("expected inherited SSH user 'testuser', got %q", tunnel.SSHUser)
	}
	if tunnel.AuthMethod != "key" {
		t.Fatalf("expected inherited auth 'key', got %q", tunnel.AuthMethod)
	}
	if tunnel.LocalHost != "127.0.0.1" {
		t.Fatalf("expected default local host '127.0.0.1', got %q", tunnel.LocalHost)
	}
	if tunnel.Type != "local" {
		t.Fatalf("expected default type 'local', got %q", tunnel.Type)
	}
}

func TestTunnelOverridesDefaults(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")

	yaml := `version: 1
defaults:
  ssh_port: 22
  ssh_user: default-user
groups:
  dev:
    tunnels:
      - name: db
        local_port: 5432
        remote_host: db.internal
        remote_port: 5432
        ssh_host: bastion
        ssh_user: override-user
        ssh_port: 3333
`
	os.WriteFile(path, []byte(yaml), 0o644)
	cfg, _ := Load(path)

	tunnel := cfg.Groups["dev"].Tunnels[0]
	if tunnel.SSHUser != "override-user" {
		t.Fatalf("expected overridden user, got %q", tunnel.SSHUser)
	}
	if tunnel.SSHPort != 3333 {
		t.Fatalf("expected overridden port 3333, got %d", tunnel.SSHPort)
	}
}

func TestAllTunnels(t *testing.T) {
	cfg := Defaults()
	cfg.Groups["dev"] = Group{
		Tunnels: []TunnelConfig{
			{Name: "t1", LocalPort: 1},
			{Name: "t2", LocalPort: 2},
		},
	}
	cfg.Groups["prod"] = Group{
		Tunnels: []TunnelConfig{
			{Name: "t3", LocalPort: 3},
		},
	}
	cfg.applyDefaults()

	all := cfg.AllTunnels()
	if len(all) != 3 {
		t.Fatalf("expected 3 tunnels, got %d", len(all))
	}
}

func TestFindTunnel(t *testing.T) {
	cfg := Defaults()
	cfg.Groups["dev"] = Group{
		Tunnels: []TunnelConfig{
			{Name: "target", LocalPort: 5432, RemoteHost: "db", RemotePort: 5432, SSHHost: "bastion"},
		},
	}
	cfg.applyDefaults()

	rt, found := cfg.FindTunnel("target")
	if !found {
		t.Fatal("tunnel not found")
	}
	if rt.Group != "dev" {
		t.Fatalf("expected group 'dev', got %q", rt.Group)
	}

	_, found = cfg.FindTunnel("nonexistent")
	if found {
		t.Fatal("expected not found for nonexistent tunnel")
	}
}

func TestTunnelsByGroup(t *testing.T) {
	cfg := Defaults()
	cfg.Groups["dev"] = Group{Tunnels: []TunnelConfig{{Name: "t1"}, {Name: "t2"}}}
	cfg.Groups["prod"] = Group{Tunnels: []TunnelConfig{{Name: "t3"}}}
	cfg.applyDefaults()

	devTunnels, ok := cfg.TunnelsByGroup("dev")
	if !ok {
		t.Fatal("group 'dev' not found")
	}
	if len(devTunnels) != 2 {
		t.Fatalf("expected 2 dev tunnels, got %d", len(devTunnels))
	}

	_, ok = cfg.TunnelsByGroup("nope")
	if ok {
		t.Fatal("expected not found for nonexistent group")
	}
}

func TestLoadOrDefaultMissingFile(t *testing.T) {
	cfg, err := LoadOrDefault("/nonexistent/path/config.yaml")
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Version != 1 {
		t.Fatalf("expected default version 1, got %d", cfg.Version)
	}
}

func TestResolvedTunnelCarriesHealthFields(t *testing.T) {
	cfg := Defaults()
	cfg.Groups["dev"] = Group{
		Tunnels: []TunnelConfig{{Name: "t1", LocalPort: 1, SSHHost: "h"}},
	}
	cfg.applyDefaults()

	all := cfg.AllTunnels()
	if len(all) != 1 {
		t.Fatal("expected 1 tunnel")
	}
	rt := all[0]
	if rt.KeepaliveInterval != cfg.Defaults.KeepaliveInterval {
		t.Fatalf("expected keepalive interval %v, got %v", cfg.Defaults.KeepaliveInterval, rt.KeepaliveInterval)
	}
	if rt.ReconnectMaxInterval != cfg.Defaults.ReconnectMaxInterval {
		t.Fatalf("expected reconnect max %v, got %v", cfg.Defaults.ReconnectMaxInterval, rt.ReconnectMaxInterval)
	}
}
