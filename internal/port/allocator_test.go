package port

import (
	"net"
	"testing"
)

func TestClaimAndRelease(t *testing.T) {
	a := NewAllocator()

	// Find a free port to test with.
	ln, _ := net.Listen("tcp", "127.0.0.1:0")
	port := ln.Addr().(*net.TCPAddr).Port
	ln.Close()

	// Claim should succeed on a free port.
	if err := a.Claim(port, "127.0.0.1", "tunnel-a"); err != nil {
		t.Fatalf("claim free port: %v", err)
	}

	// Double claim by the same tunnel should succeed.
	if err := a.Claim(port, "127.0.0.1", "tunnel-a"); err != nil {
		t.Fatalf("re-claim by same tunnel: %v", err)
	}

	// Claim by a different tunnel should fail.
	if err := a.Claim(port, "127.0.0.1", "tunnel-b"); err == nil {
		t.Fatal("expected conflict error for different tunnel")
	}

	// Release and claim by another tunnel should succeed.
	a.Release(port, "tunnel-a")
	if err := a.Claim(port, "127.0.0.1", "tunnel-b"); err != nil {
		t.Fatalf("claim after release: %v", err)
	}
}

func TestClaimPortInUse(t *testing.T) {
	a := NewAllocator()

	// Bind a port so it's in use by the OS.
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()
	port := ln.Addr().(*net.TCPAddr).Port

	// Claim should fail because the OS has it bound.
	if err := a.Claim(port, "127.0.0.1", "tunnel-a"); err == nil {
		t.Fatal("expected error for port in use by OS")
	}
}

func TestFindFreePort(t *testing.T) {
	a := NewAllocator()

	port, err := a.FindFreePort("127.0.0.1", 30000)
	if err != nil {
		t.Fatalf("FindFreePort: %v", err)
	}
	if port < 30000 || port >= 30100 {
		t.Fatalf("expected port in range 30000-30099, got %d", port)
	}
}

func TestReleaseWrongTunnel(t *testing.T) {
	a := NewAllocator()

	ln, _ := net.Listen("tcp", "127.0.0.1:0")
	port := ln.Addr().(*net.TCPAddr).Port
	ln.Close()

	a.Claim(port, "127.0.0.1", "tunnel-a")

	// Release by wrong tunnel name should be a no-op.
	a.Release(port, "tunnel-b")

	// Port should still be claimed by tunnel-a.
	if err := a.Claim(port, "127.0.0.1", "tunnel-b"); err == nil {
		t.Fatal("expected conflict — release by wrong tunnel should be no-op")
	}
}
