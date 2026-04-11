package engine

import (
	"context"
	"fmt"
	"sort"
	"sync"

	"github.com/rs/zerolog/log"

	"github.com/hyperplex-tech/bore/internal/config"
	"github.com/hyperplex-tech/bore/internal/event"
	"github.com/hyperplex-tech/bore/internal/health"
	"github.com/hyperplex-tech/bore/internal/port"
)

// Engine manages all active tunnels.
type Engine struct {
	mu      sync.RWMutex
	tunnels map[string]Tunneler // name -> tunnel
	bus     *event.Bus
	alloc   *port.Allocator
	mux     *Mux
	monitor *health.Monitor
	ctx     context.Context
	cancel  context.CancelFunc
}

// NewEngine creates a new tunnel engine.
func NewEngine(bus *event.Bus) *Engine {
	ctx, cancel := context.WithCancel(context.Background())
	return &Engine{
		tunnels: make(map[string]Tunneler),
		bus:     bus,
		alloc:   port.NewAllocator(),
		mux:     NewMux(),
		monitor: health.NewMonitor(),
		ctx:     ctx,
		cancel:  cancel,
	}
}

// newTunneler creates the appropriate tunnel implementation based on type.
func (e *Engine) newTunneler(rt config.ResolvedTunnel) Tunneler {
	if rt.Type == "k8s" {
		return NewK8sTunnel(rt, e.bus, e.alloc)
	}
	return NewTunnel(rt, e.bus, e.alloc, e.mux, e.monitor)
}

// LoadConfig registers all tunnels from the config (without connecting them).
func (e *Engine) LoadConfig(cfg *config.Config) {
	e.mu.Lock()
	defer e.mu.Unlock()

	for _, rt := range cfg.AllTunnels() {
		if _, exists := e.tunnels[rt.Name]; !exists {
			e.tunnels[rt.Name] = e.newTunneler(rt)
		}
	}
}

// Connect starts tunnels by name. If names is empty and group is set,
// connects all tunnels in that group. If both are empty, connects all.
func (e *Engine) Connect(names []string, group string, cfg *config.Config) ([]TunnelInfo, error) {
	targets, err := e.resolveTargets(names, group, cfg)
	if err != nil {
		return nil, err
	}

	var results []TunnelInfo
	for _, t := range targets {
		if err := t.Connect(e.ctx); err != nil {
			log.Error().Err(err).Str("tunnel", t.GetConfig().Name).Msg("connect failed")
		}
		results = append(results, t.Info())
	}
	return results, nil
}

// Disconnect stops tunnels by name. If names is empty and group is set,
// disconnects all tunnels in that group. If both are empty, disconnects all.
func (e *Engine) Disconnect(names []string, group string, cfg *config.Config) ([]TunnelInfo, error) {
	targets, err := e.resolveTargets(names, group, cfg)
	if err != nil {
		return nil, err
	}

	var results []TunnelInfo
	for _, t := range targets {
		t.Disconnect()
		results = append(results, t.Info())
	}
	return results, nil
}

// List returns info for all tunnels, optionally filtered by group.
func (e *Engine) List(group string) []TunnelInfo {
	e.mu.RLock()
	defer e.mu.RUnlock()

	var infos []TunnelInfo
	for _, t := range e.tunnels {
		if group != "" && t.GetConfig().Group != group {
			continue
		}
		infos = append(infos, t.Info())
	}
	sort.Slice(infos, func(i, j int) bool {
		if infos[i].Group != infos[j].Group {
			return infos[i].Group < infos[j].Group
		}
		return infos[i].Name < infos[j].Name
	})
	return infos
}

// Get returns info for a single tunnel by name.
func (e *Engine) Get(name string) (TunnelInfo, bool) {
	e.mu.RLock()
	defer e.mu.RUnlock()
	t, ok := e.tunnels[name]
	if !ok {
		return TunnelInfo{}, false
	}
	return t.Info(), true
}

// Reconcile diffs the new config against the running state:
// - New tunnels in config are registered (but not connected).
// - Tunnels removed from config are disconnected and unregistered.
// - Tunnels whose config changed are disconnected, replaced, and left stopped.
// Returns counts of (added, removed, updated) tunnels.
func (e *Engine) Reconcile(cfg *config.Config) (added, removed, updated int) {
	e.mu.Lock()
	defer e.mu.Unlock()

	// Build a set of tunnel names in the new config.
	newTunnels := make(map[string]config.ResolvedTunnel)
	for _, rt := range cfg.AllTunnels() {
		newTunnels[rt.Name] = rt
	}

	// Remove tunnels that no longer exist in config.
	for name, t := range e.tunnels {
		if _, exists := newTunnels[name]; !exists {
			t.Disconnect()
			delete(e.tunnels, name)
			removed++
			log.Info().Str("tunnel", name).Msg("reconcile: removed tunnel")
		}
	}

	// Add new tunnels and detect changes.
	for name, rt := range newTunnels {
		existing, exists := e.tunnels[name]
		if !exists {
			e.tunnels[name] = e.newTunneler(rt)
			added++
			log.Info().Str("tunnel", name).Msg("reconcile: added tunnel")
		} else if tunnelConfigChanged(existing.GetConfig(), rt) {
			existing.Disconnect()
			e.tunnels[name] = e.newTunneler(rt)
			updated++
			log.Info().Str("tunnel", name).Msg("reconcile: updated tunnel config")
		}
	}

	return
}

// tunnelConfigChanged returns true if the tunnel config has materially changed.
func tunnelConfigChanged(old config.ResolvedTunnel, new config.ResolvedTunnel) bool {
	return old.Group != new.Group ||
		old.LocalPort != new.LocalPort ||
		old.LocalHost != new.LocalHost ||
		old.RemoteHost != new.RemoteHost ||
		old.RemotePort != new.RemotePort ||
		old.SSHHost != new.SSHHost ||
		old.SSHPort != new.SSHPort ||
		old.SSHUser != new.SSHUser ||
		old.AuthMethod != new.AuthMethod ||
		old.IdentityFile != new.IdentityFile ||
		old.Type != new.Type ||
		old.K8sContext != new.K8sContext ||
		old.K8sNamespace != new.K8sNamespace ||
		old.K8sResource != new.K8sResource
}

// Shutdown disconnects all tunnels and cancels the engine context.
func (e *Engine) Shutdown() {
	e.cancel()
	e.monitor.StopAll()

	e.mu.RLock()
	for _, t := range e.tunnels {
		t.Disconnect()
	}
	e.mu.RUnlock()

	e.mux.CloseAll()
}

// ActiveCount returns the number of currently active tunnels.
func (e *Engine) ActiveCount() int {
	e.mu.RLock()
	defer e.mu.RUnlock()
	count := 0
	for _, t := range e.tunnels {
		if t.Status() == StatusActive {
			count++
		}
	}
	return count
}

// TotalCount returns the total number of registered tunnels.
func (e *Engine) TotalCount() int {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return len(e.tunnels)
}

// MuxStats returns the number of shared SSH connections and total refcount.
func (e *Engine) MuxStats() (conns, refs int) {
	return e.mux.Stats()
}

// resolveTargets finds the tunnels to operate on, ensuring they're registered
// in the engine (creating them on-the-fly from config if needed).
func (e *Engine) resolveTargets(names []string, group string, cfg *config.Config) ([]Tunneler, error) {
	e.mu.Lock()
	defer e.mu.Unlock()

	if len(names) > 0 {
		var targets []Tunneler
		for _, name := range names {
			t, ok := e.tunnels[name]
			if !ok {
				// Try to find in config and register.
				rt, found := cfg.FindTunnel(name)
				if !found {
					return nil, fmt.Errorf("tunnel %q not found", name)
				}
				t = e.newTunneler(*rt)
				e.tunnels[name] = t
			}
			targets = append(targets, t)
		}
		return targets, nil
	}

	// Resolve by group (or all).
	var resolved []config.ResolvedTunnel
	if group != "" {
		tunnels, ok := cfg.TunnelsByGroup(group)
		if !ok {
			return nil, fmt.Errorf("group %q not found", group)
		}
		resolved = tunnels
	} else {
		resolved = cfg.AllTunnels()
	}

	var targets []Tunneler
	for _, rt := range resolved {
		t, ok := e.tunnels[rt.Name]
		if !ok {
			t = e.newTunneler(rt)
			e.tunnels[rt.Name] = t
		}
		targets = append(targets, t)
	}
	return targets, nil
}
