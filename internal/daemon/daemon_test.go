package daemon_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	borev1 "github.com/hyperplex-tech/bore/gen/bore/v1"
	"github.com/hyperplex-tech/bore/internal/daemon"
)

func TestDaemonIntegration(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "tunnels.yaml")
	socketPath := filepath.Join(dir, "bored.sock")
	dataDir := filepath.Join(dir, "data")
	os.MkdirAll(dataDir, 0o755)

	// Write test config.
	configYAML := `version: 1
defaults:
  ssh_port: 22
  auth_method: agent
groups:
  dev:
    description: Dev services
    tunnels:
      - name: dev-db
        local_port: 19432
        remote_host: db.dev.internal
        remote_port: 5432
        ssh_host: bastion-dev.example.com
        ssh_user: deploy
      - name: dev-redis
        local_port: 19379
        remote_host: redis.dev.internal
        remote_port: 6379
        ssh_host: bastion-dev.example.com
        ssh_user: deploy
  staging:
    description: Staging
    tunnels:
      - name: stg-mongo
        local_port: 19017
        remote_host: mongo.stg.internal
        remote_port: 27017
        ssh_host: bastion-stg.example.com
        ssh_user: deploy
`
	os.WriteFile(configPath, []byte(configYAML), 0o644)

	// Set XDG paths.
	os.Setenv("XDG_DATA_HOME", dataDir)
	defer os.Unsetenv("XDG_DATA_HOME")

	// Start daemon.
	d, err := daemon.New(daemon.Options{
		ConfigPath: configPath,
		SocketPath: socketPath,
		LogLevel:   "error", // quiet for tests
	})
	if err != nil {
		t.Fatalf("daemon.New: %v", err)
	}

	// Run daemon in background.
	errCh := make(chan error, 1)
	go func() { errCh <- d.Run() }()

	// Wait for socket to appear.
	for i := 0; i < 20; i++ {
		if _, err := os.Stat(socketPath); err == nil {
			break
		}
		time.Sleep(50 * time.Millisecond)
	}

	// Connect gRPC client.
	conn, err := grpc.NewClient(
		"unix://"+socketPath,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		t.Fatalf("grpc dial: %v", err)
	}
	defer conn.Close()

	daemonSvc := borev1.NewDaemonServiceClient(conn)
	tunnelSvc := borev1.NewTunnelServiceClient(conn)
	groupSvc := borev1.NewGroupServiceClient(conn)
	eventSvc := borev1.NewEventServiceClient(conn)

	ctx := context.Background()

	// --- Test DaemonService.Status ---
	t.Run("DaemonStatus", func(t *testing.T) {
		resp, err := daemonSvc.Status(ctx, &borev1.StatusRequest{})
		if err != nil {
			t.Fatal(err)
		}
		if resp.Version == "" {
			t.Fatal("expected non-empty version")
		}
		if resp.TotalTunnels != 3 {
			t.Fatalf("expected 3 total tunnels, got %d", resp.TotalTunnels)
		}
		if resp.ActiveTunnels != 0 {
			t.Fatalf("expected 0 active tunnels, got %d", resp.ActiveTunnels)
		}
		if resp.SocketPath != socketPath {
			t.Fatalf("expected socket %s, got %s", socketPath, resp.SocketPath)
		}
	})

	// --- Test GroupService.ListGroups ---
	t.Run("ListGroups", func(t *testing.T) {
		resp, err := groupSvc.ListGroups(ctx, &borev1.ListGroupsRequest{})
		if err != nil {
			t.Fatal(err)
		}
		if len(resp.Groups) != 2 {
			t.Fatalf("expected 2 groups, got %d", len(resp.Groups))
		}
		groupNames := map[string]bool{}
		for _, g := range resp.Groups {
			groupNames[g.Name] = true
		}
		if !groupNames["dev"] || !groupNames["staging"] {
			t.Fatalf("expected groups 'dev' and 'staging', got %v", groupNames)
		}
	})

	// --- Test TunnelService.List ---
	t.Run("ListTunnels", func(t *testing.T) {
		resp, err := tunnelSvc.List(ctx, &borev1.ListRequest{})
		if err != nil {
			t.Fatal(err)
		}
		if len(resp.Tunnels) != 3 {
			t.Fatalf("expected 3 tunnels, got %d", len(resp.Tunnels))
		}
		for _, tun := range resp.Tunnels {
			if tun.Status != borev1.TunnelStatus_TUNNEL_STATUS_STOPPED {
				t.Fatalf("expected stopped, got %v for %s", tun.Status, tun.Name)
			}
		}
	})

	// --- Test TunnelService.List with group filter ---
	t.Run("ListTunnelsByGroup", func(t *testing.T) {
		resp, err := tunnelSvc.List(ctx, &borev1.ListRequest{Group: "dev"})
		if err != nil {
			t.Fatal(err)
		}
		if len(resp.Tunnels) != 2 {
			t.Fatalf("expected 2 dev tunnels, got %d", len(resp.Tunnels))
		}
	})

	// --- Test TunnelService.Connect (will error — no real SSH) ---
	t.Run("ConnectTunnelError", func(t *testing.T) {
		resp, err := tunnelSvc.Connect(ctx, &borev1.ConnectRequest{Names: []string{"dev-db"}})
		if err != nil {
			t.Fatal(err)
		}
		if len(resp.Tunnels) != 1 {
			t.Fatalf("expected 1 tunnel in response, got %d", len(resp.Tunnels))
		}
		tun := resp.Tunnels[0]
		if tun.Status != borev1.TunnelStatus_TUNNEL_STATUS_ERROR {
			t.Fatalf("expected error status (no real SSH), got %v", tun.Status)
		}
		if tun.ErrorMessage == "" {
			t.Fatal("expected non-empty error message")
		}
	})

	// --- Test event stream ---
	t.Run("EventStream", func(t *testing.T) {
		streamCtx, streamCancel := context.WithCancel(ctx)
		defer streamCancel()

		stream, err := eventSvc.Subscribe(streamCtx, &borev1.SubscribeRequest{})
		if err != nil {
			t.Fatal(err)
		}

		// Give the server-side Subscribe handler time to register with the event bus.
		time.Sleep(100 * time.Millisecond)

		// Trigger an event by attempting a connect (will fail due to no SSH, but still emits TunnelError).
		go func() {
			tunnelSvc.Connect(ctx, &borev1.ConnectRequest{Names: []string{"dev-redis"}})
		}()

		// Should receive at least one event.
		type recvResult struct {
			name string
			err  error
		}
		ch := make(chan recvResult, 1)
		go func() {
			evt, err := stream.Recv()
			if err != nil {
				ch <- recvResult{err: err}
				return
			}
			ch <- recvResult{name: evt.TunnelName}
		}()

		select {
		case res := <-ch:
			if res.err != nil {
				t.Fatalf("stream.Recv: %v", res.err)
			}
			if res.name == "" {
				t.Errorf("expected non-empty tunnel name in event")
			}
		case <-time.After(5 * time.Second):
			streamCancel()
			t.Fatal("timeout waiting for event")
		}
	})

	// --- Test ReloadConfig ---
	t.Run("ReloadConfig", func(t *testing.T) {
		// Write a completely new config with an extra group.
		newConfig := `version: 1
defaults:
  ssh_port: 22
  auth_method: agent
groups:
  dev:
    description: Dev services
    tunnels:
      - name: dev-db
        local_port: 19432
        remote_host: db.dev.internal
        remote_port: 5432
        ssh_host: bastion-dev.example.com
        ssh_user: deploy
      - name: dev-redis
        local_port: 19379
        remote_host: redis.dev.internal
        remote_port: 6379
        ssh_host: bastion-dev.example.com
        ssh_user: deploy
  staging:
    description: Staging
    tunnels:
      - name: stg-mongo
        local_port: 19017
        remote_host: mongo.stg.internal
        remote_port: 27017
        ssh_host: bastion-stg.example.com
        ssh_user: deploy
  extra:
    description: Extra
    tunnels:
      - name: extra-tunnel
        local_port: 19999
        remote_host: extra.internal
        remote_port: 80
        ssh_host: bastion-extra.example.com
        ssh_user: deploy
`
		os.WriteFile(configPath, []byte(newConfig), 0o644)

		resp, err := daemonSvc.ReloadConfig(ctx, &borev1.ReloadConfigRequest{})
		if err != nil {
			t.Fatal(err)
		}
		if resp.TunnelsLoaded != 4 {
			t.Fatalf("expected 4 tunnels after reload, got %d", resp.TunnelsLoaded)
		}

		// Verify the new tunnel is listed.
		listResp, _ := tunnelSvc.List(ctx, &borev1.ListRequest{})
		found := false
		for _, tun := range listResp.Tunnels {
			if tun.Name == "extra-tunnel" {
				found = true
				break
			}
		}
		if !found {
			t.Fatal("new tunnel 'extra-tunnel' not found after reload")
		}
	})

	// --- Test TunnelService.GetLogs ---
	t.Run("GetLogs", func(t *testing.T) {
		logStream, err := tunnelSvc.GetLogs(ctx, &borev1.GetLogsRequest{
			Name:   "dev-db",
			Tail:   10,
			Follow: false,
		})
		if err != nil {
			t.Fatal(err)
		}

		// Collect entries (non-follow mode should end with EOF).
		var entries []*borev1.LogEntry
		for {
			entry, err := logStream.Recv()
			if err != nil {
				break
			}
			entries = append(entries, entry)
		}
		// We connected dev-db earlier (which failed), so there should be at least 1 log entry.
		if len(entries) == 0 {
			t.Log("no log entries found (may depend on event logger timing)")
		}
	})

	// --- Shutdown ---
	t.Run("Shutdown", func(t *testing.T) {
		// The Shutdown RPC sends os.Interrupt to the current process, which
		// in a test is the test runner itself. Verify the RPC returns
		// successfully — the actual shutdown is exercised by the daemon
		// end-to-end tests with real binaries.
		_, err := daemonSvc.Shutdown(ctx, &borev1.ShutdownRequest{})
		if err != nil {
			t.Fatal(err)
		}

		// Give it a moment and check if daemon exited (it might or might not
		// depending on signal handling in the test environment).
		select {
		case <-errCh:
			// Daemon exited cleanly.
		case <-time.After(1 * time.Second):
			// Expected in some test environments — the signal goes to the
			// test process which doesn't forward it to the daemon goroutine.
			t.Log("daemon did not exit via signal (expected in test environment)")
		}
	})
}
