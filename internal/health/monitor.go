package health

import (
	"context"
	"sync"
	"time"

	"github.com/rs/zerolog/log"
	"golang.org/x/crypto/ssh"
)

// State represents the health state of a monitored connection.
type State int

const (
	StateHealthy State = iota
	StateDegraded
	StateDead
)

// Check is a callback invoked when health state changes.
type Check struct {
	// OnDead is called when the connection is declared dead after
	// maxFailures consecutive keepalive failures.
	OnDead func()

	// OnRecovered is called when a previously-degraded connection responds
	// to a keepalive again.
	OnRecovered func()
}

// Monitor watches SSH connections with keepalive requests.
type Monitor struct {
	mu       sync.Mutex
	watchers map[string]*watcher
	closeCh  chan struct{}
}

// watcher tracks a single SSH connection's health.
type watcher struct {
	client      *ssh.Client
	interval    time.Duration
	maxFailures int
	check       Check
	cancel      context.CancelFunc

	failures int
	state    State
}

// NewMonitor creates a new health monitor.
func NewMonitor() *Monitor {
	return &Monitor{
		watchers: make(map[string]*watcher),
		closeCh:  make(chan struct{}),
	}
}

// Watch starts monitoring an SSH connection. The name should be unique
// (typically the tunnel name). Cancels any existing watch for the same name.
func (m *Monitor) Watch(name string, client *ssh.Client, interval time.Duration, maxFailures int, check Check) {
	if interval <= 0 {
		interval = 30 * time.Second
	}
	if maxFailures <= 0 {
		maxFailures = 3
	}

	m.mu.Lock()
	// Cancel existing watcher for this name.
	if w, ok := m.watchers[name]; ok {
		w.cancel()
	}

	ctx, cancel := context.WithCancel(context.Background())
	w := &watcher{
		client:      client,
		interval:    interval,
		maxFailures: maxFailures,
		check:       check,
		cancel:      cancel,
		state:       StateHealthy,
	}
	m.watchers[name] = w
	m.mu.Unlock()

	go w.run(ctx, name)
}

// Unwatch stops monitoring a connection.
func (m *Monitor) Unwatch(name string) {
	m.mu.Lock()
	if w, ok := m.watchers[name]; ok {
		w.cancel()
		delete(m.watchers, name)
	}
	m.mu.Unlock()
}

// StopAll cancels all watchers.
func (m *Monitor) StopAll() {
	close(m.closeCh)

	m.mu.Lock()
	defer m.mu.Unlock()
	for name, w := range m.watchers {
		w.cancel()
		delete(m.watchers, name)
	}
}

func (w *watcher) run(ctx context.Context, name string) {
	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()

	logger := log.With().Str("tunnel", name).Logger()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			ok := w.sendKeepalive()
			if ok {
				if w.state == StateDegraded {
					logger.Info().Msg("health: connection recovered")
					w.state = StateHealthy
					w.failures = 0
					if w.check.OnRecovered != nil {
						w.check.OnRecovered()
					}
				} else {
					w.failures = 0
				}
			} else {
				w.failures++
				logger.Warn().
					Int("failures", w.failures).
					Int("max", w.maxFailures).
					Msg("health: keepalive failed")

				if w.failures >= w.maxFailures {
					if w.state != StateDead {
						logger.Error().Msg("health: connection declared dead")
						w.state = StateDead
						if w.check.OnDead != nil {
							w.check.OnDead()
						}
						return // Stop monitoring — reconnect logic takes over.
					}
				} else {
					w.state = StateDegraded
				}
			}
		}
	}
}

// sendKeepalive sends an SSH keepalive request and returns true if it gets a response.
func (w *watcher) sendKeepalive() bool {
	_, _, err := w.client.SendRequest("keepalive@bore", true, nil)
	return err == nil
}
