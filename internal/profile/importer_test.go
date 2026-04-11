package profile

import (
	"os"
	"path/filepath"
	"testing"
)

func TestImportSSHConfig(t *testing.T) {
	dir := t.TempDir()
	sshConfig := filepath.Join(dir, "config")

	content := `Host bastion-dev
    HostName bastion-dev.example.com
    User deploy
    Port 2222
    IdentityFile ~/.ssh/id_ed25519

Host db-tunnel
    HostName bastion-dev.example.com
    User deploy
    ProxyJump jump1.example.com,jump2.example.com
    LocalForward 5432 db.internal:5432
    LocalForward 6379 redis.internal:6379

Host web-server
    HostName web.example.com
    User admin
    LocalForward 8080 localhost:80

Host wildcard-*
    User root
    LocalForward 9999 localhost:9999
`
	if err := os.WriteFile(sshConfig, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	hosts, err := ImportSSHConfig(sshConfig)
	if err != nil {
		t.Fatalf("ImportSSHConfig: %v", err)
	}

	// Should find db-tunnel (2 forwards) and web-server (1 forward).
	// bastion-dev has no LocalForward, wildcard is skipped.
	if len(hosts) != 2 {
		t.Fatalf("expected 2 hosts with LocalForward, got %d", len(hosts))
	}

	// db-tunnel
	dbHost := findHost(hosts, "db-tunnel")
	if dbHost == nil {
		t.Fatal("db-tunnel not found")
	}
	if dbHost.HostName != "bastion-dev.example.com" {
		t.Errorf("expected HostName bastion-dev.example.com, got %s", dbHost.HostName)
	}
	if dbHost.User != "deploy" {
		t.Errorf("expected User deploy, got %s", dbHost.User)
	}
	// ProxyJump values that are already real hostnames should be left as-is.
	if dbHost.ProxyJump != "jump1.example.com,jump2.example.com" {
		t.Errorf("unexpected ProxyJump: %s", dbHost.ProxyJump)
	}
	if len(dbHost.LocalForwards) != 2 {
		t.Fatalf("expected 2 LocalForwards, got %d", len(dbHost.LocalForwards))
	}
	if dbHost.LocalForwards[0].LocalPort != 5432 || dbHost.LocalForwards[0].RemoteHost != "db.internal" {
		t.Errorf("unexpected first forward: %+v", dbHost.LocalForwards[0])
	}

	// web-server
	webHost := findHost(hosts, "web-server")
	if webHost == nil {
		t.Fatal("web-server not found")
	}
	if len(webHost.LocalForwards) != 1 {
		t.Fatalf("expected 1 LocalForward, got %d", len(webHost.LocalForwards))
	}
	if webHost.LocalForwards[0].LocalPort != 8080 || webHost.LocalForwards[0].RemoteHost != "localhost" {
		t.Errorf("unexpected forward: %+v", webHost.LocalForwards[0])
	}
}

func TestImportSSHConfigProxyJumpAliasResolution(t *testing.T) {
	dir := t.TempDir()
	sshConfig := filepath.Join(dir, "config")

	content := `Host or-vpn01
    HostName vpn01.openrecovery.com
    User vpnuser
    Port 2222

Host or-tunnel-dev
    HostName recoveryiqdev.openrecovery.com
    User dev
    IdentityFile ~/.ssh/openrecovery-infra-dev.pem
    ProxyJump or-vpn01
    LocalForward 3306 localhost:3306

Host multi-jump-tunnel
    HostName db.internal.example.com
    User admin
    ProxyJump or-vpn01,bastion02
    LocalForward 5432 db.internal:5432

Host bastion02
    HostName bastion02.example.com
    User jump
`
	if err := os.WriteFile(sshConfig, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	hosts, err := ImportSSHConfig(sshConfig)
	if err != nil {
		t.Fatalf("ImportSSHConfig: %v", err)
	}

	if len(hosts) != 2 {
		t.Fatalf("expected 2 hosts with LocalForward, got %d", len(hosts))
	}

	// or-tunnel-dev: ProxyJump "or-vpn01" should resolve to "vpnuser@vpn01.openrecovery.com:2222"
	devHost := findHost(hosts, "or-tunnel-dev")
	if devHost == nil {
		t.Fatal("or-tunnel-dev not found")
	}
	if devHost.ProxyJump != "vpnuser@vpn01.openrecovery.com:2222" {
		t.Errorf("expected resolved ProxyJump 'vpnuser@vpn01.openrecovery.com:2222', got '%s'", devHost.ProxyJump)
	}

	// multi-jump-tunnel: "or-vpn01,bastion02" should resolve both aliases.
	mjHost := findHost(hosts, "multi-jump-tunnel")
	if mjHost == nil {
		t.Fatal("multi-jump-tunnel not found")
	}
	if mjHost.ProxyJump != "vpnuser@vpn01.openrecovery.com:2222,jump@bastion02.example.com" {
		t.Errorf("expected resolved ProxyJump 'vpnuser@vpn01.openrecovery.com:2222,jump@bastion02.example.com', got '%s'", mjHost.ProxyJump)
	}

	// Verify the tunnel configs also get the resolved jump hosts.
	tunnels := ToTunnelConfigs(hosts)
	for _, tc := range tunnels {
		if tc.Name == "or-tunnel-dev" {
			if len(tc.JumpHosts) != 1 || tc.JumpHosts[0] != "vpnuser@vpn01.openrecovery.com:2222" {
				t.Errorf("expected resolved jump host, got %v", tc.JumpHosts)
			}
		}
	}
}

func TestToTunnelConfigs(t *testing.T) {
	hosts := []SSHHost{
		{
			Alias:    "db-tunnel",
			HostName: "bastion.example.com",
			User:     "deploy",
			Port:     22,
			ProxyJump: "jump1.example.com",
			LocalForwards: []LocalForward{
				{LocalHost: "127.0.0.1", LocalPort: 5432, RemoteHost: "db.internal", RemotePort: 5432},
				{LocalHost: "127.0.0.1", LocalPort: 6379, RemoteHost: "redis.internal", RemotePort: 6379},
			},
		},
	}

	tunnels := ToTunnelConfigs(hosts)
	if len(tunnels) != 2 {
		t.Fatalf("expected 2 tunnels, got %d", len(tunnels))
	}

	// Multiple forwards from same host get numbered names.
	if tunnels[0].Name != "db-tunnel-1" {
		t.Errorf("expected name db-tunnel-1, got %s", tunnels[0].Name)
	}
	if tunnels[1].Name != "db-tunnel-2" {
		t.Errorf("expected name db-tunnel-2, got %s", tunnels[1].Name)
	}
	if len(tunnels[0].JumpHosts) != 1 || tunnels[0].JumpHosts[0] != "jump1.example.com" {
		t.Errorf("unexpected jump hosts: %v", tunnels[0].JumpHosts)
	}
}

func TestImportSSHConfigMissing(t *testing.T) {
	_, err := ImportSSHConfig("/nonexistent/path")
	if err == nil {
		t.Fatal("expected error for missing file")
	}
}

func findHost(hosts []SSHHost, alias string) *SSHHost {
	for i := range hosts {
		if hosts[i].Alias == alias {
			return &hosts[i]
		}
	}
	return nil
}
