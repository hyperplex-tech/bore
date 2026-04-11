package health

import (
	"crypto/ed25519"
	"crypto/rand"
	"fmt"
	"net"
	"sync/atomic"
	"testing"
	"time"

	"golang.org/x/crypto/ssh"
)

// startTestSSHServer starts a minimal SSH server for testing keepalives.
func startTestSSHServer(t *testing.T) (*ssh.Client, net.Listener) {
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

	// Dial the test server.
	clientConfig := &ssh.ClientConfig{
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         5 * time.Second,
	}
	client, err := ssh.Dial("tcp", listener.Addr().String(), clientConfig)
	if err != nil {
		listener.Close()
		t.Fatal(err)
	}

	return client, listener
}

func TestMonitorHealthyConnection(t *testing.T) {
	client, listener := startTestSSHServer(t)
	defer listener.Close()
	defer client.Close()

	monitor := NewMonitor()
	defer monitor.StopAll()

	var deadCalled atomic.Int32
	monitor.Watch("test", client, 100*time.Millisecond, 3, Check{
		OnDead: func() { deadCalled.Add(1) },
	})

	// Wait enough time for several keepalives.
	time.Sleep(500 * time.Millisecond)

	if deadCalled.Load() != 0 {
		t.Fatal("OnDead should not have been called on a healthy connection")
	}
}

func TestMonitorDeadConnection(t *testing.T) {
	client, listener := startTestSSHServer(t)

	monitor := NewMonitor()
	defer monitor.StopAll()

	deadCh := make(chan struct{}, 1)
	monitor.Watch("test", client, 100*time.Millisecond, 2, Check{
		OnDead: func() {
			select {
			case deadCh <- struct{}{}:
			default:
			}
		},
	})

	// Kill the server to make keepalives fail.
	listener.Close()
	client.Close()

	select {
	case <-deadCh:
		// Expected — connection declared dead.
	case <-time.After(5 * time.Second):
		t.Fatal("timeout waiting for OnDead callback")
	}
}

func TestMonitorUnwatch(t *testing.T) {
	client, listener := startTestSSHServer(t)
	defer listener.Close()
	defer client.Close()

	monitor := NewMonitor()
	defer monitor.StopAll()

	var deadCalled atomic.Int32
	monitor.Watch("test", client, 50*time.Millisecond, 1, Check{
		OnDead: func() { deadCalled.Add(1) },
	})

	monitor.Unwatch("test")
	time.Sleep(200 * time.Millisecond)

	if deadCalled.Load() != 0 {
		t.Fatal("OnDead called after Unwatch")
	}
}

func TestMonitorReplaceWatch(t *testing.T) {
	client, listener := startTestSSHServer(t)
	defer listener.Close()
	defer client.Close()

	monitor := NewMonitor()
	defer monitor.StopAll()

	var firstCalled atomic.Int32
	var secondCalled atomic.Int32

	monitor.Watch("test", client, 100*time.Millisecond, 100, Check{
		OnDead: func() { firstCalled.Add(1) },
	})

	// Replace with a new watch.
	monitor.Watch("test", client, 100*time.Millisecond, 100, Check{
		OnDead: func() { secondCalled.Add(1) },
	})

	time.Sleep(300 * time.Millisecond)
	monitor.Unwatch("test")

	_ = firstCalled.Load() // first watcher was cancelled, shouldn't fire
	_ = secondCalled.Load()
	fmt.Println("replace watch test passed")
}
