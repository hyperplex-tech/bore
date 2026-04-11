package engine

import (
	"fmt"
	"sync"
	"time"

	"github.com/rs/zerolog/log"
	"golang.org/x/crypto/ssh"
)

const (
	// muxGracePeriod is how long an idle SSH connection stays open after its
	// last tunnel disconnects, in case a new tunnel reuses it soon.
	muxGracePeriod = 30 * time.Second
)

// muxEntry is a reference-counted SSH connection.
type muxEntry struct {
	client  *ssh.Client
	refs    int
	key     string
	graceAt time.Time // set when refs drops to 0
}

// Mux shares SSH connections among tunnels targeting the same host.
// Key format: "user@host:port".
type Mux struct {
	mu      sync.Mutex
	conns   map[string]*muxEntry
	closeCh chan struct{}
}

// NewMux creates a new SSH connection multiplexer.
func NewMux() *Mux {
	m := &Mux{
		conns:   make(map[string]*muxEntry),
		closeCh: make(chan struct{}),
	}
	go m.reapLoop()
	return m
}

// muxKey builds the map key for an SSH connection.
func muxKey(user, host string, port int) string {
	return fmt.Sprintf("%s@%s:%d", user, host, port)
}

// Acquire returns an existing shared SSH connection or dials a new one.
// The caller must call Release when done.
func (m *Mux) Acquire(user, host string, port int, dialFn func() (*ssh.Client, error)) (*ssh.Client, error) {
	key := muxKey(user, host, port)

	m.mu.Lock()
	if entry, ok := m.conns[key]; ok {
		// Check if the connection is still alive.
		_, _, err := entry.client.SendRequest("keepalive@bore", true, nil)
		if err == nil {
			entry.refs++
			entry.graceAt = time.Time{} // clear grace timer
			m.mu.Unlock()
			log.Debug().Str("key", key).Int("refs", entry.refs).Msg("mux: reusing SSH connection")
			return entry.client, nil
		}
		// Dead connection — remove and dial fresh.
		log.Debug().Str("key", key).Msg("mux: stale connection, re-dialing")
		entry.client.Close()
		delete(m.conns, key)
	}
	m.mu.Unlock()

	// Dial outside the lock to avoid blocking other tunnels.
	client, err := dialFn()
	if err != nil {
		return nil, err
	}

	m.mu.Lock()
	// Another goroutine may have raced and inserted — check again.
	if entry, ok := m.conns[key]; ok {
		// Someone else won the race. Close our new client and reuse theirs.
		client.Close()
		entry.refs++
		entry.graceAt = time.Time{}
		m.mu.Unlock()
		return entry.client, nil
	}

	m.conns[key] = &muxEntry{
		client: client,
		refs:   1,
		key:    key,
	}
	m.mu.Unlock()

	log.Debug().Str("key", key).Msg("mux: new SSH connection")
	return client, nil
}

// Release decrements the refcount. When it hits 0, the connection enters a
// grace period before being closed.
func (m *Mux) Release(user, host string, port int) {
	key := muxKey(user, host, port)

	m.mu.Lock()
	defer m.mu.Unlock()

	entry, ok := m.conns[key]
	if !ok {
		return
	}

	entry.refs--
	if entry.refs <= 0 {
		entry.refs = 0
		entry.graceAt = time.Now().Add(muxGracePeriod)
		log.Debug().Str("key", key).Dur("grace", muxGracePeriod).Msg("mux: connection idle, grace period started")
	}
}

// CloseAll forcibly closes all SSH connections.
func (m *Mux) CloseAll() {
	close(m.closeCh)

	m.mu.Lock()
	defer m.mu.Unlock()

	for key, entry := range m.conns {
		entry.client.Close()
		delete(m.conns, key)
	}
}

// Stats returns the number of shared SSH connections and total refcount.
func (m *Mux) Stats() (conns, refs int) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, entry := range m.conns {
		conns++
		refs += entry.refs
	}
	return
}

// reapLoop periodically closes idle connections whose grace period has expired.
func (m *Mux) reapLoop() {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-m.closeCh:
			return
		case now := <-ticker.C:
			m.mu.Lock()
			for key, entry := range m.conns {
				if entry.refs == 0 && !entry.graceAt.IsZero() && now.After(entry.graceAt) {
					log.Debug().Str("key", key).Msg("mux: closing idle SSH connection")
					entry.client.Close()
					delete(m.conns, key)
				}
			}
			m.mu.Unlock()
		}
	}
}
