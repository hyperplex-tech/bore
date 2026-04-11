package engine

import (
	"context"
	"fmt"
	"net"
	"sync"
	"sync/atomic"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"golang.org/x/crypto/ssh"

	"github.com/hyperplex-tech/bore/internal/auth"
	"github.com/hyperplex-tech/bore/internal/config"
	"github.com/hyperplex-tech/bore/internal/event"
	"github.com/hyperplex-tech/bore/internal/health"
	"github.com/hyperplex-tech/bore/internal/hook"
	"github.com/hyperplex-tech/bore/internal/port"
)

// TunnelStatus represents the current state of a tunnel.
type TunnelStatus string

const (
	StatusStopped    TunnelStatus = "stopped"
	StatusConnecting TunnelStatus = "connecting"
	StatusActive     TunnelStatus = "active"
	StatusError      TunnelStatus = "error"
	StatusPaused     TunnelStatus = "paused"
	StatusRetrying   TunnelStatus = "retrying"
)

// Tunneler is the common interface for all tunnel types (SSH, K8s, etc.).
type Tunneler interface {
	Connect(ctx context.Context) error
	Disconnect()
	Info() TunnelInfo
	Status() TunnelStatus
	GetConfig() config.ResolvedTunnel
}

// GetConfig returns the tunnel's resolved configuration.
func (t *Tunnel) GetConfig() config.ResolvedTunnel {
	return t.Config
}

// Tunnel manages the lifecycle of a single SSH port-forward.
type Tunnel struct {
	Config config.ResolvedTunnel

	mu            sync.RWMutex
	status        TunnelStatus
	sshClient     *ssh.Client
	listener      net.Listener
	connectedAt   time.Time
	lastErrorAt   time.Time
	errorMessage  string
	retryCount    int
	nextRetrySecs int
	activeConns   atomic.Int64

	cancel  context.CancelFunc
	bus     *event.Bus
	alloc   *port.Allocator
	mux     *Mux
	monitor *health.Monitor
	backoff *health.Backoff
	log     zerolog.Logger
}

// NewTunnel creates a new tunnel from a resolved config.
func NewTunnel(cfg config.ResolvedTunnel, bus *event.Bus, alloc *port.Allocator, mux *Mux, monitor *health.Monitor) *Tunnel {
	reconnectMax := cfg.ReconnectMaxInterval
	if reconnectMax <= 0 {
		reconnectMax = 60 * time.Second
	}
	return &Tunnel{
		Config:  cfg,
		status:  StatusStopped,
		bus:     bus,
		alloc:   alloc,
		mux:     mux,
		monitor: monitor,
		backoff: health.NewBackoff(reconnectMax),
		log: log.With().
			Str("tunnel", cfg.Name).
			Str("group", cfg.Group).
			Logger(),
	}
}

// Status returns the current tunnel status.
func (t *Tunnel) Status() TunnelStatus {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.status
}

// Info returns a snapshot of the tunnel's current state.
type TunnelInfo struct {
	Name          string
	Group         string
	Status        TunnelStatus
	LocalHost     string
	LocalPort     int
	RemoteHost    string
	RemotePort    int
	SSHHost       string
	SSHPort       int
	SSHUser       string
	ConnectedAt   time.Time
	LastErrorAt   time.Time
	ErrorMessage  string
	RetryCount    int
	NextRetrySecs int
	ActiveConns   int64
	Config        config.TunnelConfig
}

func (t *Tunnel) Info() TunnelInfo {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return TunnelInfo{
		Name:          t.Config.Name,
		Group:         t.Config.Group,
		Status:        t.status,
		LocalHost:     t.Config.LocalHost,
		LocalPort:     t.Config.LocalPort,
		RemoteHost:    t.Config.RemoteHost,
		RemotePort:    t.Config.RemotePort,
		SSHHost:       t.Config.SSHHost,
		SSHPort:       t.Config.SSHPort,
		SSHUser:       t.Config.SSHUser,
		ConnectedAt:   t.connectedAt,
		LastErrorAt:   t.lastErrorAt,
		ErrorMessage:  t.errorMessage,
		RetryCount:    t.retryCount,
		NextRetrySecs: t.nextRetrySecs,
		ActiveConns:   t.activeConns.Load(),
		Config:        t.Config.TunnelConfig,
	}
}

// Connect starts the tunnel: dials SSH (via mux), binds the appropriate port, starts forwarding.
func (t *Tunnel) Connect(ctx context.Context) error {
	t.mu.Lock()
	if t.status == StatusActive || t.status == StatusConnecting {
		t.mu.Unlock()
		return nil
	}
	t.status = StatusConnecting
	t.errorMessage = ""
	t.nextRetrySecs = 0
	ctx, t.cancel = context.WithCancel(ctx)
	t.mu.Unlock()

	t.log.Info().Msg("connecting")

	isRemote := t.Config.Type == "remote"

	// Claim the local port (only for non-remote tunnels that listen locally).
	if !isRemote {
		if err := t.alloc.Claim(t.Config.LocalPort, t.Config.LocalHost, t.Config.Name); err != nil {
			t.setError(fmt.Errorf("port claim: %w", err))
			return err
		}
	}

	// Run pre-connect hook.
	if err := t.runHook("pre_connect", "connecting"); err != nil {
		if !isRemote {
			t.alloc.Release(t.Config.LocalPort, t.Config.Name)
		}
		t.setError(fmt.Errorf("pre_connect hook: %w", err))
		return err
	}

	// Resolve auth. When an identity file is specified without an explicit
	// auth method, use key-only auth to avoid "Too many authentication failures"
	// from SSH agent key enumeration (matches IdentitiesOnly behavior).
	authMethod := t.Config.AuthMethod
	if authMethod == "" && t.Config.IdentityFile != "" {
		authMethod = "key"
	}
	provider := auth.NewProvider(authMethod)
	authMethods, err := provider.AuthMethods(auth.AuthConfig{
		Method:       authMethod,
		IdentityFile: t.Config.IdentityFile,
		SSHUser:      t.Config.SSHUser,
	})
	if err != nil {
		if !isRemote {
			t.alloc.Release(t.Config.LocalPort, t.Config.Name)
		}
		t.setError(fmt.Errorf("auth: %w", err))
		return err
	}

	// Dial SSH via the multiplexer.
	sshAddr := fmt.Sprintf("%s:%d", t.Config.SSHHost, t.Config.SSHPort)
	sshConfig := &ssh.ClientConfig{
		User:            t.Config.SSHUser,
		Auth:            authMethods,
		HostKeyCallback: ssh.InsecureIgnoreHostKey(), // TODO: proper host key verification
		Timeout:         15 * time.Second,
	}

	sshClient, err := t.mux.Acquire(t.Config.SSHUser, t.Config.SSHHost, t.Config.SSHPort, func() (*ssh.Client, error) {
		if len(t.Config.JumpHosts) > 0 {
			return dialSSHViaJumpHosts(t.Config.JumpHosts, sshAddr, sshConfig)
		}
		return ssh.Dial("tcp", sshAddr, sshConfig)
	})
	if err != nil {
		if !isRemote {
			t.alloc.Release(t.Config.LocalPort, t.Config.Name)
		}
		t.setError(fmt.Errorf("SSH dial %s: %w", sshAddr, err))
		return err
	}

	listenAddr := fmt.Sprintf("%s:%d", t.Config.LocalHost, t.Config.LocalPort)
	remoteAddr := fmt.Sprintf("%s:%d", t.Config.RemoteHost, t.Config.RemotePort)

	var listener net.Listener
	if isRemote {
		// Remote forward: listen on the remote SSH server, forward back to local.
		listener, err = sshClient.Listen("tcp", remoteAddr)
		if err != nil {
			t.mux.Release(t.Config.SSHUser, t.Config.SSHHost, t.Config.SSHPort)
			t.setError(fmt.Errorf("remote listen %s: %w", remoteAddr, err))
			return err
		}
	} else {
		// Local/dynamic forward: bind local port.
		listener, err = net.Listen("tcp", listenAddr)
		if err != nil {
			t.mux.Release(t.Config.SSHUser, t.Config.SSHHost, t.Config.SSHPort)
			t.alloc.Release(t.Config.LocalPort, t.Config.Name)
			t.setError(fmt.Errorf("listen %s: %w", listenAddr, err))
			return err
		}
	}

	t.mu.Lock()
	t.sshClient = sshClient
	t.listener = listener
	t.status = StatusActive
	t.connectedAt = time.Now()
	t.retryCount = 0
	t.backoff.Reset()
	t.mu.Unlock()

	t.log.Info().
		Str("local", listenAddr).
		Str("remote", remoteAddr).
		Str("via", sshAddr).
		Msg("tunnel active")

	t.bus.Publish(event.Event{
		Type:       event.TunnelConnected,
		TunnelName: t.Config.Name,
		Message:    fmt.Sprintf("%s → %s via %s", listenAddr, remoteAddr, t.Config.SSHHost),
	})

	// Run post-connect hook (non-blocking — log errors but don't fail the tunnel).
	if err := t.runHook("post_connect", "connected"); err != nil {
		t.log.Warn().Err(err).Msg("post_connect hook failed")
	}

	// Register health monitoring.
	reconnect := t.Config.Reconnect != nil && *t.Config.Reconnect
	t.monitor.Watch(t.Config.Name, sshClient, t.Config.KeepaliveInterval, t.Config.KeepaliveMaxFailures, health.Check{
		OnDead: func() {
			t.log.Warn().Msg("SSH connection dead, tearing down tunnel")
			t.teardown()
			if reconnect {
				go t.reconnectLoop(ctx)
			}
		},
	})

	// Accept loop in a goroutine — dispatch based on tunnel type.
	switch t.Config.Type {
	case "dynamic":
		go serveSocks5(listener, sshClient, t.Config.Name)
	case "remote":
		go t.reverseAcceptLoop(ctx, listener, listenAddr)
	default:
		go t.acceptLoop(ctx, sshClient, listener, remoteAddr)
	}

	return nil
}

// Disconnect tears down the tunnel and stops reconnection.
func (t *Tunnel) Disconnect() {
	t.mu.Lock()
	if t.status == StatusStopped {
		t.mu.Unlock()
		return
	}
	cancel := t.cancel
	t.mu.Unlock()

	t.log.Info().Msg("disconnecting")

	// Cancel context first to stop reconnect loop.
	if cancel != nil {
		cancel()
	}

	t.monitor.Unwatch(t.Config.Name)
	t.teardown()

	t.mu.Lock()
	t.status = StatusStopped
	t.mu.Unlock()

	t.bus.Publish(event.Event{
		Type:       event.TunnelDisconnected,
		TunnelName: t.Config.Name,
	})
}

// teardown closes the listener and releases the SSH connection, but does NOT
// set status to stopped (caller decides the next state).
func (t *Tunnel) teardown() {
	t.mu.Lock()
	listener := t.listener
	t.listener = nil
	t.sshClient = nil
	t.mu.Unlock()

	if listener != nil {
		listener.Close()
	}
	t.mux.Release(t.Config.SSHUser, t.Config.SSHHost, t.Config.SSHPort)
	// Remote forwards don't claim a local port.
	if t.Config.Type != "remote" {
		t.alloc.Release(t.Config.LocalPort, t.Config.Name)
	}
}

// reconnectLoop attempts to re-establish the tunnel with exponential backoff.
func (t *Tunnel) reconnectLoop(ctx context.Context) {
	for {
		delay := t.backoff.Next()
		nextSecs := int(delay.Seconds())

		t.mu.Lock()
		t.status = StatusRetrying
		t.retryCount++
		t.nextRetrySecs = nextSecs
		t.mu.Unlock()

		t.log.Info().
			Int("attempt", t.backoff.Attempt()).
			Dur("delay", delay).
			Msg("scheduling reconnect")

		t.bus.Publish(event.Event{
			Type:       event.TunnelRetrying,
			TunnelName: t.Config.Name,
			Message:    fmt.Sprintf("retry in %ds (attempt %d)", nextSecs, t.backoff.Attempt()),
		})

		select {
		case <-ctx.Done():
			return // Tunnel was disconnected or daemon is shutting down.
		case <-time.After(delay):
		}

		// Check if we were disconnected while waiting.
		select {
		case <-ctx.Done():
			return
		default:
		}

		t.log.Info().Int("attempt", t.backoff.Attempt()).Msg("reconnecting")

		// Reset state for a fresh Connect attempt.
		t.mu.Lock()
		t.status = StatusConnecting
		t.errorMessage = ""
		t.nextRetrySecs = 0
		t.mu.Unlock()

		err := t.doConnect(ctx)
		if err == nil {
			t.log.Info().Msg("reconnected successfully")
			return // Success — health monitor will take over again.
		}

		t.log.Warn().Err(err).Msg("reconnect failed")
		// Loop will try again with increased backoff.
	}
}

// doConnect is the inner connect logic (reused by both Connect and reconnectLoop).
func (t *Tunnel) doConnect(ctx context.Context) error {
	isRemote := t.Config.Type == "remote"

	// Claim the local port (only for non-remote tunnels that listen locally).
	if !isRemote {
		if err := t.alloc.Claim(t.Config.LocalPort, t.Config.LocalHost, t.Config.Name); err != nil {
			t.setError(fmt.Errorf("port claim: %w", err))
			return err
		}
	}

	// Resolve auth. When an identity file is specified without an explicit
	// auth method, use key-only auth to avoid "Too many authentication failures"
	// from SSH agent key enumeration (matches IdentitiesOnly behavior).
	authMethod := t.Config.AuthMethod
	if authMethod == "" && t.Config.IdentityFile != "" {
		authMethod = "key"
	}
	provider := auth.NewProvider(authMethod)
	authMethods, err := provider.AuthMethods(auth.AuthConfig{
		Method:       authMethod,
		IdentityFile: t.Config.IdentityFile,
		SSHUser:      t.Config.SSHUser,
	})
	if err != nil {
		if !isRemote {
			t.alloc.Release(t.Config.LocalPort, t.Config.Name)
		}
		t.setError(fmt.Errorf("auth: %w", err))
		return err
	}

	sshAddr := fmt.Sprintf("%s:%d", t.Config.SSHHost, t.Config.SSHPort)
	sshConfig := &ssh.ClientConfig{
		User:            t.Config.SSHUser,
		Auth:            authMethods,
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         15 * time.Second,
	}

	sshClient, err := t.mux.Acquire(t.Config.SSHUser, t.Config.SSHHost, t.Config.SSHPort, func() (*ssh.Client, error) {
		if len(t.Config.JumpHosts) > 0 {
			return dialSSHViaJumpHosts(t.Config.JumpHosts, sshAddr, sshConfig)
		}
		return ssh.Dial("tcp", sshAddr, sshConfig)
	})
	if err != nil {
		if !isRemote {
			t.alloc.Release(t.Config.LocalPort, t.Config.Name)
		}
		t.setError(fmt.Errorf("SSH dial %s: %w", sshAddr, err))
		return err
	}

	listenAddr := fmt.Sprintf("%s:%d", t.Config.LocalHost, t.Config.LocalPort)
	remoteAddr := fmt.Sprintf("%s:%d", t.Config.RemoteHost, t.Config.RemotePort)

	var listener net.Listener
	if isRemote {
		listener, err = sshClient.Listen("tcp", remoteAddr)
		if err != nil {
			t.mux.Release(t.Config.SSHUser, t.Config.SSHHost, t.Config.SSHPort)
			t.setError(fmt.Errorf("remote listen %s: %w", remoteAddr, err))
			return err
		}
	} else {
		listener, err = net.Listen("tcp", listenAddr)
		if err != nil {
			t.mux.Release(t.Config.SSHUser, t.Config.SSHHost, t.Config.SSHPort)
			t.alloc.Release(t.Config.LocalPort, t.Config.Name)
			t.setError(fmt.Errorf("listen %s: %w", listenAddr, err))
			return err
		}
	}

	t.mu.Lock()
	t.sshClient = sshClient
	t.listener = listener
	t.status = StatusActive
	t.connectedAt = time.Now()
	t.backoff.Reset()
	t.nextRetrySecs = 0
	t.mu.Unlock()

	t.log.Info().
		Str("local", listenAddr).
		Str("remote", remoteAddr).
		Str("via", sshAddr).
		Msg("tunnel active")

	t.bus.Publish(event.Event{
		Type:       event.TunnelConnected,
		TunnelName: t.Config.Name,
		Message:    fmt.Sprintf("%s → %s via %s (reconnected)", listenAddr, remoteAddr, t.Config.SSHHost),
	})

	// Re-register health monitoring.
	reconnect := t.Config.Reconnect != nil && *t.Config.Reconnect
	t.monitor.Watch(t.Config.Name, sshClient, t.Config.KeepaliveInterval, t.Config.KeepaliveMaxFailures, health.Check{
		OnDead: func() {
			t.log.Warn().Msg("SSH connection dead, tearing down tunnel")
			t.teardown()
			if reconnect {
				go t.reconnectLoop(ctx)
			}
		},
	})

	switch t.Config.Type {
	case "dynamic":
		go serveSocks5(listener, sshClient, t.Config.Name)
	case "remote":
		go t.reverseAcceptLoop(ctx, listener, listenAddr)
	default:
		go t.acceptLoop(ctx, sshClient, listener, remoteAddr)
	}

	return nil
}

func (t *Tunnel) acceptLoop(ctx context.Context, sshClient *ssh.Client, listener net.Listener, remoteAddr string) {
	for {
		conn, err := listener.Accept()
		if err != nil {
			select {
			case <-ctx.Done():
				return // Normal shutdown.
			default:
			}
			t.log.Debug().Err(err).Msg("accept failed")
			// Don't setError here if the listener was closed by teardown —
			// the reconnect loop handles state transitions.
			return
		}

		t.activeConns.Add(1)
		go func() {
			defer t.activeConns.Add(-1)
			forward(conn, sshClient, remoteAddr)
		}()
	}
}

// reverseAcceptLoop accepts connections from the remote SSH listener and
// forwards them to the local address. This is the accept loop for remote
// forwards (ssh -R equivalent).
func (t *Tunnel) reverseAcceptLoop(ctx context.Context, listener net.Listener, localAddr string) {
	for {
		remoteConn, err := listener.Accept()
		if err != nil {
			select {
			case <-ctx.Done():
				return
			default:
			}
			t.log.Debug().Err(err).Msg("remote accept failed")
			return
		}

		t.activeConns.Add(1)
		go func() {
			defer t.activeConns.Add(-1)
			forwardReverse(remoteConn, localAddr)
		}()
	}
}

// runHook executes a pre/post-connect hook if configured.
func (t *Tunnel) runHook(phase, status string) error {
	if t.Config.Hooks == nil {
		return nil
	}
	var command string
	switch phase {
	case "pre_connect":
		command = t.Config.Hooks.PreConnect
	case "post_connect":
		command = t.Config.Hooks.PostConnect
	}
	if command == "" {
		return nil
	}
	return hook.Run(command, hook.Env{
		TunnelName: t.Config.Name,
		Group:      t.Config.Group,
		LocalHost:  t.Config.LocalHost,
		LocalPort:  t.Config.LocalPort,
		RemoteHost: t.Config.RemoteHost,
		RemotePort: t.Config.RemotePort,
		SSHHost:    t.Config.SSHHost,
		SSHPort:    t.Config.SSHPort,
		SSHUser:    t.Config.SSHUser,
		Status:     status,
	})
}

func (t *Tunnel) setError(err error) {
	t.mu.Lock()
	t.status = StatusError
	t.errorMessage = err.Error()
	t.lastErrorAt = time.Now()
	t.mu.Unlock()

	t.log.Error().Err(err).Msg("tunnel error")

	t.bus.Publish(event.Event{
		Type:       event.TunnelError,
		TunnelName: t.Config.Name,
		Message:    err.Error(),
	})
}
