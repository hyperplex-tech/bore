package engine_test

import (
	"fmt"
	"io"
	"net"
	"testing"
	"time"

	"github.com/hyperplex-tech/bore/internal/config"
	"github.com/hyperplex-tech/bore/internal/engine"
	"github.com/hyperplex-tech/bore/internal/event"
)

func TestMultipleTunnelsShareMux(t *testing.T) {
	// 1. Start test SSH server.
	sshAddr := startTestSSHServer(t)
	sshHost, sshPortStr, _ := net.SplitHostPort(sshAddr)
	var sshPort int
	fmt.Sscanf(sshPortStr, "%d", &sshPort)

	// 2. Start two echo servers (simulating two remote services).
	echoHost1, echoPort1 := startEchoServer(t)
	echoHost2, echoPort2 := startEchoServer(t)

	// 3. Find two free local ports.
	ln1, _ := net.Listen("tcp", "127.0.0.1:0")
	localPort1 := ln1.Addr().(*net.TCPAddr).Port
	ln1.Close()
	ln2, _ := net.Listen("tcp", "127.0.0.1:0")
	localPort2 := ln2.Addr().(*net.TCPAddr).Port
	ln2.Close()

	// 4. Write the test key.
	keyFile := t.TempDir() + "/test_key"
	writeTestKey(t, keyFile)

	reconnect := false
	cfg := &config.Config{
		Version:  1,
		Defaults: config.Defaults().Defaults,
		Groups: map[string]config.Group{
			"test": {
				Tunnels: []config.TunnelConfig{
					{
						Name:         "tunnel-1",
						Type:         "local",
						LocalHost:    "127.0.0.1",
						LocalPort:    localPort1,
						RemoteHost:   echoHost1,
						RemotePort:   echoPort1,
						SSHHost:      sshHost,
						SSHPort:      sshPort,
						SSHUser:      "testuser",
						AuthMethod:   "key",
						IdentityFile: keyFile,
						Reconnect:    &reconnect,
					},
					{
						Name:         "tunnel-2",
						Type:         "local",
						LocalHost:    "127.0.0.1",
						LocalPort:    localPort2,
						RemoteHost:   echoHost2,
						RemotePort:   echoPort2,
						SSHHost:      sshHost,
						SSHPort:      sshPort,
						SSHUser:      "testuser",
						AuthMethod:   "key",
						IdentityFile: keyFile,
						Reconnect:    &reconnect,
					},
				},
			},
		},
	}

	bus := event.NewBus()
	eng := engine.NewEngine(bus)
	eng.LoadConfig(cfg)

	// 5. Connect both tunnels — they should share the same SSH connection.
	infos, err := eng.Connect(nil, "test", cfg)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	if len(infos) != 2 {
		t.Fatalf("expected 2 tunnels, got %d", len(infos))
	}
	for _, info := range infos {
		if info.Status != engine.StatusActive {
			t.Fatalf("tunnel %s: expected active, got %s (error: %s)", info.Name, info.Status, info.ErrorMessage)
		}
	}

	// 6. Verify mux is sharing: should be 1 SSH connection with 2 refs.
	muxConns, muxRefs := eng.MuxStats()
	if muxConns != 1 {
		t.Fatalf("expected 1 shared SSH connection, got %d", muxConns)
	}
	if muxRefs != 2 {
		t.Fatalf("expected 2 mux refs, got %d", muxRefs)
	}

	// 7. Verify both tunnels work independently.
	for i, port := range []int{localPort1, localPort2} {
		conn, err := net.DialTimeout("tcp", fmt.Sprintf("127.0.0.1:%d", port), 2*time.Second)
		if err != nil {
			t.Fatalf("tunnel-%d dial: %v", i+1, err)
		}
		msg := []byte(fmt.Sprintf("hello tunnel %d", i+1))
		conn.Write(msg)
		buf := make([]byte, len(msg))
		conn.SetReadDeadline(time.Now().Add(5 * time.Second))
		io.ReadFull(conn, buf)
		if string(buf) != string(msg) {
			t.Fatalf("tunnel-%d echo mismatch: got %q, want %q", i+1, buf, msg)
		}
		conn.Close()
	}

	// 8. Disconnect one tunnel — mux should still have 1 ref.
	eng.Disconnect([]string{"tunnel-1"}, "", cfg)
	muxConns, muxRefs = eng.MuxStats()
	if muxConns != 1 {
		t.Fatalf("after disconnect 1: expected 1 conn, got %d", muxConns)
	}
	if muxRefs != 1 {
		t.Fatalf("after disconnect 1: expected 1 ref, got %d", muxRefs)
	}

	// 9. Tunnel-2 should still work.
	conn, err := net.DialTimeout("tcp", fmt.Sprintf("127.0.0.1:%d", localPort2), 2*time.Second)
	if err != nil {
		t.Fatalf("tunnel-2 still alive: %v", err)
	}
	msg := []byte("still alive")
	conn.Write(msg)
	buf := make([]byte, len(msg))
	conn.SetReadDeadline(time.Now().Add(5 * time.Second))
	io.ReadFull(conn, buf)
	if string(buf) != string(msg) {
		t.Fatalf("tunnel-2 echo mismatch after disconnect-1")
	}
	conn.Close()

	// 10. Shutdown.
	eng.Shutdown()
	_, muxRefs = eng.MuxStats()
	if muxRefs != 0 {
		t.Fatalf("after shutdown: expected 0 refs, got %d", muxRefs)
	}
}
