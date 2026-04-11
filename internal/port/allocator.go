package port

import (
	"fmt"
	"net"
	"sync"
)

// Allocator tracks port claims and detects conflicts.
type Allocator struct {
	mu     sync.Mutex
	claims map[int]string // port -> tunnel name
}

// NewAllocator creates a new port allocator.
func NewAllocator() *Allocator {
	return &Allocator{
		claims: make(map[int]string),
	}
}

// Claim reserves a local port for a tunnel. Returns an error if the port is
// already claimed by another tunnel or is in use by another process.
func (a *Allocator) Claim(port int, host, tunnelName string) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	// Check if already claimed by another bore tunnel.
	if owner, ok := a.claims[port]; ok && owner != tunnelName {
		return fmt.Errorf("port %d already claimed by tunnel %q", port, owner)
	}

	// Check if port is in use by another process on the system.
	if err := checkPortAvailable(host, port); err != nil {
		return fmt.Errorf("port %d on %s is already in use: %w", port, host, err)
	}

	a.claims[port] = tunnelName
	return nil
}

// Release frees a previously claimed port.
func (a *Allocator) Release(port int, tunnelName string) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if owner, ok := a.claims[port]; ok && owner == tunnelName {
		delete(a.claims, port)
	}
}

// FindFreePort finds an available port starting from the given port.
func (a *Allocator) FindFreePort(host string, startPort int) (int, error) {
	a.mu.Lock()
	defer a.mu.Unlock()

	for port := startPort; port < startPort+100; port++ {
		if _, claimed := a.claims[port]; claimed {
			continue
		}
		if err := checkPortAvailable(host, port); err == nil {
			return port, nil
		}
	}
	return 0, fmt.Errorf("no free port found in range %d-%d", startPort, startPort+99)
}

// checkPortAvailable tests if a port can be bound.
func checkPortAvailable(host string, port int) error {
	addr := fmt.Sprintf("%s:%d", host, port)
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}
	ln.Close()
	return nil
}
