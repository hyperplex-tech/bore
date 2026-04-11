package engine_test

import (
	"bytes"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/pem"
	"fmt"
	"io"
	"net"
	"os"
	"testing"
	"time"

	"golang.org/x/crypto/ssh"

	"github.com/hyperplex-tech/bore/internal/config"
	"github.com/hyperplex-tech/bore/internal/engine"
	"github.com/hyperplex-tech/bore/internal/event"
)

var testClientSigner ssh.Signer
var testClientPrivKey ed25519.PrivateKey

// startTestSSHServer creates a minimal SSH server that handles direct-tcpip
// channel requests (port forwarding). Returns the listening address.
func startTestSSHServer(t *testing.T) string {
	t.Helper()

	// Generate a host key.
	_, hostPriv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	hostSigner, err := ssh.NewSignerFromKey(hostPriv)
	if err != nil {
		t.Fatal(err)
	}

	// Generate a client key.
	_, clientPriv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	clientSigner, err := ssh.NewSignerFromKey(clientPriv)
	if err != nil {
		t.Fatal(err)
	}
	testClientSigner = clientSigner
	testClientPrivKey = clientPriv
	clientPub := clientSigner.PublicKey()

	serverConfig := &ssh.ServerConfig{
		PublicKeyCallback: func(conn ssh.ConnMetadata, key ssh.PublicKey) (*ssh.Permissions, error) {
			if bytes.Equal(key.Marshal(), clientPub.Marshal()) {
				return nil, nil
			}
			return nil, fmt.Errorf("unknown public key")
		},
	}
	serverConfig.AddHostKey(hostSigner)

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}

	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				return
			}
			go handleSSHConn(conn, serverConfig)
		}
	}()

	t.Cleanup(func() { listener.Close() })
	return listener.Addr().String()
}

func handleSSHConn(nConn net.Conn, config *ssh.ServerConfig) {
	sshConn, chans, reqs, err := ssh.NewServerConn(nConn, config)
	if err != nil {
		nConn.Close()
		return
	}
	defer sshConn.Close()

	go ssh.DiscardRequests(reqs)

	for newCh := range chans {
		if newCh.ChannelType() != "direct-tcpip" {
			newCh.Reject(ssh.UnknownChannelType, "unsupported")
			continue
		}

		type directTCPIP struct {
			DestHost string
			DestPort uint32
			SrcHost  string
			SrcPort  uint32
		}
		var payload directTCPIP
		if err := ssh.Unmarshal(newCh.ExtraData(), &payload); err != nil {
			newCh.Reject(ssh.ConnectionFailed, "bad payload")
			continue
		}

		ch, reqs, err := newCh.Accept()
		if err != nil {
			continue
		}
		go ssh.DiscardRequests(reqs)

		target := net.JoinHostPort(payload.DestHost, fmt.Sprintf("%d", payload.DestPort))
		targetConn, err := net.DialTimeout("tcp", target, 5*time.Second)
		if err != nil {
			ch.Close()
			continue
		}

		go func() {
			defer ch.Close()
			defer targetConn.Close()
			go io.Copy(ch, targetConn)
			io.Copy(targetConn, ch)
		}()
	}
}

// startEchoServer starts a TCP server that echoes back whatever it receives.
func startEchoServer(t *testing.T) (string, int) {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}

	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				return
			}
			go func(c net.Conn) {
				defer c.Close()
				io.Copy(c, c)
			}(conn)
		}
	}()

	t.Cleanup(func() { listener.Close() })
	addr := listener.Addr().(*net.TCPAddr)
	return addr.IP.String(), addr.Port
}

func writeTestKey(t *testing.T, path string) {
	t.Helper()
	block, err := ssh.MarshalPrivateKey(testClientPrivKey, "")
	if err != nil {
		t.Fatal(err)
	}
	pemBytes := pem.EncodeToMemory(block)
	if err := os.WriteFile(path, pemBytes, 0o600); err != nil {
		t.Fatal(err)
	}
}

func TestTunnelEndToEnd(t *testing.T) {
	// 1. Start test SSH server.
	sshAddr := startTestSSHServer(t)
	sshHost, sshPortStr, _ := net.SplitHostPort(sshAddr)
	var sshPort int
	fmt.Sscanf(sshPortStr, "%d", &sshPort)

	// 2. Start echo server (the "remote" service).
	echoHost, echoPort := startEchoServer(t)

	// 3. Find a free local port.
	ln, _ := net.Listen("tcp", "127.0.0.1:0")
	localPort := ln.Addr().(*net.TCPAddr).Port
	ln.Close()

	// 4. Write the test client key to a temp file.
	keyFile := t.TempDir() + "/test_key"
	writeTestKey(t, keyFile)

	// 5. Build config.
	cfg := &config.Config{
		Version:  1,
		Defaults: config.Defaults().Defaults,
		Groups: map[string]config.Group{
			"test": {
				Description: "integration test",
				Tunnels: []config.TunnelConfig{
					{
						Name:         "echo-tunnel",
						Type:         "local",
						LocalHost:    "127.0.0.1",
						LocalPort:    localPort,
						RemoteHost:   echoHost,
						RemotePort:   echoPort,
						SSHHost:      sshHost,
						SSHPort:      sshPort,
						SSHUser:      "testuser",
						AuthMethod:   "key",
						IdentityFile: keyFile,
					},
				},
			},
		},
	}

	bus := event.NewBus()
	eng := engine.NewEngine(bus)
	eng.LoadConfig(cfg)

	// Subscribe to events.
	_, evtCh := bus.Subscribe(16)

	// 6. Connect.
	infos, err := eng.Connect(nil, "test", cfg)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	if len(infos) != 1 {
		t.Fatalf("expected 1 tunnel, got %d", len(infos))
	}
	if infos[0].Status != engine.StatusActive {
		t.Fatalf("expected active, got %s (error: %s)", infos[0].Status, infos[0].ErrorMessage)
	}

	// 7. Verify connected event.
	select {
	case evt := <-evtCh:
		if evt.Type != event.TunnelConnected {
			t.Fatalf("expected TunnelConnected, got %v", evt.Type)
		}
		if evt.TunnelName != "echo-tunnel" {
			t.Fatalf("expected tunnel name echo-tunnel, got %s", evt.TunnelName)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for connected event")
	}

	// 8. Send data through the tunnel and verify echo.
	conn, err := net.Dial("tcp", fmt.Sprintf("127.0.0.1:%d", localPort))
	if err != nil {
		t.Fatalf("dial tunnel: %v", err)
	}
	defer conn.Close()

	msg := []byte("hello bore tunnel!")
	if _, err := conn.Write(msg); err != nil {
		t.Fatalf("write: %v", err)
	}

	buf := make([]byte, len(msg))
	conn.SetReadDeadline(time.Now().Add(5 * time.Second))
	if _, err := io.ReadFull(conn, buf); err != nil {
		t.Fatalf("read: %v", err)
	}
	if string(buf) != string(msg) {
		t.Fatalf("echo mismatch: got %q, want %q", buf, msg)
	}

	// 9. Verify engine state.
	if eng.ActiveCount() != 1 {
		t.Fatalf("expected 1 active, got %d", eng.ActiveCount())
	}
	info, ok := eng.Get("echo-tunnel")
	if !ok {
		t.Fatal("tunnel not found")
	}
	if info.Status != engine.StatusActive {
		t.Fatalf("expected active, got %s", info.Status)
	}
	// Our conn is still open, so there should be 1 active forwarded connection.
	// (This might be 0 if the echo already completed; skip this check.)

	// 10. Disconnect.
	conn.Close()
	eng.Shutdown()
	if eng.ActiveCount() != 0 {
		t.Fatalf("expected 0 active after shutdown, got %d", eng.ActiveCount())
	}
}

func TestTunnelConnectError(t *testing.T) {
	// Test connecting to a non-existent SSH server.
	ln, _ := net.Listen("tcp", "127.0.0.1:0")
	localPort := ln.Addr().(*net.TCPAddr).Port
	ln.Close()

	cfg := &config.Config{
		Version:  1,
		Defaults: config.Defaults().Defaults,
		Groups: map[string]config.Group{
			"test": {
				Tunnels: []config.TunnelConfig{
					{
						Name:       "bad-tunnel",
						Type:       "local",
						LocalHost:  "127.0.0.1",
						LocalPort:  localPort,
						RemoteHost: "127.0.0.1",
						RemotePort: 9999,
						SSHHost:    "127.0.0.1",
						SSHPort:    1, // nothing on port 1
						SSHUser:    "nobody",
						AuthMethod: "key",
					},
				},
			},
		},
	}

	bus := event.NewBus()
	eng := engine.NewEngine(bus)
	eng.LoadConfig(cfg)

	infos, err := eng.Connect(nil, "test", cfg)
	if err != nil {
		t.Fatalf("connect returned error: %v", err)
	}
	// The tunnel should be in error state, not active.
	if len(infos) != 1 {
		t.Fatalf("expected 1 info, got %d", len(infos))
	}
	if infos[0].Status != engine.StatusError {
		t.Fatalf("expected error status, got %s", infos[0].Status)
	}
	if infos[0].ErrorMessage == "" {
		t.Fatal("expected non-empty error message")
	}

	eng.Shutdown()
}
