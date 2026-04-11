package engine

import (
	"crypto/ed25519"
	"crypto/rand"
	"fmt"
	"net"
	"testing"
	"time"

	"golang.org/x/crypto/ssh"
)

func startMuxTestSSHServer(t *testing.T) (string, func()) {
	t.Helper()

	_, hostPriv, _ := ed25519.GenerateKey(rand.Reader)
	hostSigner, _ := ssh.NewSignerFromKey(hostPriv)

	serverConfig := &ssh.ServerConfig{NoClientAuth: true}
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
			go func() {
				sshConn, chans, reqs, err := ssh.NewServerConn(conn, serverConfig)
				if err != nil {
					conn.Close()
					return
				}
				defer sshConn.Close()
				go ssh.DiscardRequests(reqs)
				for ch := range chans {
					ch.Reject(ssh.UnknownChannelType, "not supported")
				}
			}()
		}
	}()

	return listener.Addr().String(), func() { listener.Close() }
}

func TestMuxSharesConnections(t *testing.T) {
	addr, cleanup := startMuxTestSSHServer(t)
	defer cleanup()

	host, portStr, _ := net.SplitHostPort(addr)
	var port int
	_, _ = fmt.Sscanf(portStr, "%d", &port)

	mux := NewMux()
	defer mux.CloseAll()

	dialFn := func() (*ssh.Client, error) {
		return ssh.Dial("tcp", addr, &ssh.ClientConfig{
			HostKeyCallback: ssh.InsecureIgnoreHostKey(),
			Timeout:         5 * time.Second,
		})
	}

	// Acquire twice — should reuse the same connection.
	c1, err := mux.Acquire("user", host, port, dialFn)
	if err != nil {
		t.Fatal(err)
	}
	c2, err := mux.Acquire("user", host, port, dialFn)
	if err != nil {
		t.Fatal(err)
	}

	if c1 != c2 {
		t.Fatal("expected same SSH client for same endpoint")
	}

	conns, refs := mux.Stats()
	if conns != 1 || refs != 2 {
		t.Fatalf("expected 1 conn 2 refs, got %d conns %d refs", conns, refs)
	}

	// Release once — still 1 ref.
	mux.Release("user", host, port)
	_, refs = mux.Stats()
	if refs != 1 {
		t.Fatalf("expected 1 ref after one release, got %d", refs)
	}

	// Release again — enters grace period (refs=0).
	mux.Release("user", host, port)
	conns, refs = mux.Stats()
	if refs != 0 {
		t.Fatalf("expected 0 refs, got %d", refs)
	}
	// Connection still exists (grace period).
	if conns != 1 {
		t.Fatalf("expected 1 conn in grace period, got %d", conns)
	}
}

func TestMuxDifferentEndpoints(t *testing.T) {
	addr1, cleanup1 := startMuxTestSSHServer(t)
	defer cleanup1()
	addr2, cleanup2 := startMuxTestSSHServer(t)
	defer cleanup2()

	mux := NewMux()
	defer mux.CloseAll()

	dialTo := func(addr string) func() (*ssh.Client, error) {
		return func() (*ssh.Client, error) {
			return ssh.Dial("tcp", addr, &ssh.ClientConfig{
				HostKeyCallback: ssh.InsecureIgnoreHostKey(),
				Timeout:         5 * time.Second,
			})
		}
	}

	host1, portStr1, _ := net.SplitHostPort(addr1)
	var port1 int
	_, _ = fmt.Sscanf(portStr1, "%d", &port1)

	host2, portStr2, _ := net.SplitHostPort(addr2)
	var port2 int
	_, _ = fmt.Sscanf(portStr2, "%d", &port2)

	c1, err := mux.Acquire("user", host1, port1, dialTo(addr1))
	if err != nil {
		t.Fatal(err)
	}
	c2, err := mux.Acquire("user", host2, port2, dialTo(addr2))
	if err != nil {
		t.Fatal(err)
	}

	if c1 == c2 {
		t.Fatal("different endpoints should produce different clients")
	}

	conns, _ := mux.Stats()
	if conns != 2 {
		t.Fatalf("expected 2 conns, got %d", conns)
	}
}
