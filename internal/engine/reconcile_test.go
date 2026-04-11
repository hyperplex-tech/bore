package engine_test

import (
	"testing"

	"github.com/hyperplex-tech/bore/internal/config"
	"github.com/hyperplex-tech/bore/internal/engine"
	"github.com/hyperplex-tech/bore/internal/event"
)

func TestReconcileAddsNewTunnels(t *testing.T) {
	bus := event.NewBus()
	eng := engine.NewEngine(bus)

	cfg := &config.Config{
		Version:  1,
		Defaults: config.Defaults().Defaults,
		Groups: map[string]config.Group{
			"dev": {Tunnels: []config.TunnelConfig{
				{Name: "t1", LocalPort: 10001, RemoteHost: "h", RemotePort: 1, SSHHost: "s"},
			}},
		},
	}
	eng.LoadConfig(cfg)

	if eng.TotalCount() != 1 {
		t.Fatalf("expected 1 tunnel, got %d", eng.TotalCount())
	}

	// Add a new tunnel via reconcile.
	cfg.Groups["dev"] = config.Group{Tunnels: []config.TunnelConfig{
		{Name: "t1", LocalPort: 10001, RemoteHost: "h", RemotePort: 1, SSHHost: "s"},
		{Name: "t2", LocalPort: 10002, RemoteHost: "h", RemotePort: 2, SSHHost: "s"},
	}}

	added, removed, updated := eng.Reconcile(cfg)
	if added != 1 || removed != 0 || updated != 0 {
		t.Fatalf("expected +1 -0 ~0, got +%d -%d ~%d", added, removed, updated)
	}
	if eng.TotalCount() != 2 {
		t.Fatalf("expected 2 tunnels, got %d", eng.TotalCount())
	}
}

func TestReconcileRemovesTunnels(t *testing.T) {
	bus := event.NewBus()
	eng := engine.NewEngine(bus)

	cfg := &config.Config{
		Version:  1,
		Defaults: config.Defaults().Defaults,
		Groups: map[string]config.Group{
			"dev": {Tunnels: []config.TunnelConfig{
				{Name: "t1", LocalPort: 10001, RemoteHost: "h", RemotePort: 1, SSHHost: "s"},
				{Name: "t2", LocalPort: 10002, RemoteHost: "h", RemotePort: 2, SSHHost: "s"},
			}},
		},
	}
	eng.LoadConfig(cfg)

	// Remove t2.
	cfg.Groups["dev"] = config.Group{Tunnels: []config.TunnelConfig{
		{Name: "t1", LocalPort: 10001, RemoteHost: "h", RemotePort: 1, SSHHost: "s"},
	}}

	added, removed, updated := eng.Reconcile(cfg)
	if added != 0 || removed != 1 || updated != 0 {
		t.Fatalf("expected +0 -1 ~0, got +%d -%d ~%d", added, removed, updated)
	}
	if eng.TotalCount() != 1 {
		t.Fatalf("expected 1 tunnel, got %d", eng.TotalCount())
	}
}

func TestReconcileDetectsChanges(t *testing.T) {
	bus := event.NewBus()
	eng := engine.NewEngine(bus)

	cfg := &config.Config{
		Version:  1,
		Defaults: config.Defaults().Defaults,
		Groups: map[string]config.Group{
			"dev": {Tunnels: []config.TunnelConfig{
				{Name: "t1", LocalPort: 10001, RemoteHost: "h", RemotePort: 1, SSHHost: "s"},
			}},
		},
	}
	eng.LoadConfig(cfg)

	// Change the port.
	cfg.Groups["dev"] = config.Group{Tunnels: []config.TunnelConfig{
		{Name: "t1", LocalPort: 10099, RemoteHost: "h", RemotePort: 1, SSHHost: "s"},
	}}

	added, removed, updated := eng.Reconcile(cfg)
	if added != 0 || removed != 0 || updated != 1 {
		t.Fatalf("expected +0 -0 ~1, got +%d -%d ~%d", added, removed, updated)
	}
}

func TestReconcileNoChanges(t *testing.T) {
	bus := event.NewBus()
	eng := engine.NewEngine(bus)

	cfg := &config.Config{
		Version:  1,
		Defaults: config.Defaults().Defaults,
		Groups: map[string]config.Group{
			"dev": {Tunnels: []config.TunnelConfig{
				{Name: "t1", LocalPort: 10001, RemoteHost: "h", RemotePort: 1, SSHHost: "s"},
			}},
		},
	}
	eng.LoadConfig(cfg)

	added, removed, updated := eng.Reconcile(cfg)
	if added != 0 || removed != 0 || updated != 0 {
		t.Fatalf("expected +0 -0 ~0, got +%d -%d ~%d", added, removed, updated)
	}
}
