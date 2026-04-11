package engine

import (
	"bufio"
	"context"
	"fmt"
	"net"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/rs/zerolog/log"

	"github.com/hyperplex-tech/bore/internal/config"
	"github.com/hyperplex-tech/bore/internal/event"
	"github.com/hyperplex-tech/bore/internal/port"
)

// K8sTunnel wraps a `kubectl port-forward` process with the same lifecycle
// interface as an SSH Tunnel.
type K8sTunnel struct {
	Config config.ResolvedTunnel

	mu           sync.RWMutex
	status       TunnelStatus
	errorMessage string
	connectedAt  time.Time
	lastErrorAt  time.Time
	cmd          *exec.Cmd
	cancel       context.CancelFunc

	bus   *event.Bus
	alloc *port.Allocator
}

// NewK8sTunnel creates a new K8s port-forward tunnel.
func NewK8sTunnel(cfg config.ResolvedTunnel, bus *event.Bus, alloc *port.Allocator) *K8sTunnel {
	return &K8sTunnel{
		Config: cfg,
		status: StatusStopped,
		bus:    bus,
		alloc:  alloc,
	}
}

// GetConfig returns the tunnel's resolved configuration.
func (k *K8sTunnel) GetConfig() config.ResolvedTunnel {
	return k.Config
}

// Status returns the current tunnel status.
func (k *K8sTunnel) Status() TunnelStatus {
	k.mu.RLock()
	defer k.mu.RUnlock()
	return k.status
}

// Info returns a snapshot of the tunnel state.
func (k *K8sTunnel) Info() TunnelInfo {
	k.mu.RLock()
	defer k.mu.RUnlock()
	return TunnelInfo{
		Name:         k.Config.Name,
		Group:        k.Config.Group,
		Status:       k.status,
		LocalHost:    k.Config.LocalHost,
		LocalPort:    k.Config.LocalPort,
		RemoteHost:   k.Config.K8sResource,
		RemotePort:   k.Config.RemotePort,
		SSHHost:      k.Config.K8sContext,
		ErrorMessage: k.errorMessage,
		ConnectedAt:  k.connectedAt,
		LastErrorAt:  k.lastErrorAt,
		Config:       k.Config.TunnelConfig,
	}
}

// Connect starts the kubectl port-forward process.
func (k *K8sTunnel) Connect(ctx context.Context) error {
	k.mu.Lock()
	if k.status == StatusActive {
		k.mu.Unlock()
		return nil
	}
	k.status = StatusConnecting
	k.errorMessage = ""
	k.mu.Unlock()

	// Claim the local port.
	if err := k.alloc.Claim(k.Config.LocalPort, k.Config.LocalHost, k.Config.Name); err != nil {
		k.setError(fmt.Errorf("port claim: %w", err))
		return err
	}

	ctx, k.cancel = context.WithCancel(ctx)

	// Build kubectl command.
	args := []string{"port-forward"}
	if k.Config.K8sContext != "" {
		args = append(args, "--context", k.Config.K8sContext)
	}
	if k.Config.K8sNamespace != "" {
		args = append(args, "-n", k.Config.K8sNamespace)
	}
	args = append(args, k.Config.K8sResource)
	args = append(args, fmt.Sprintf("%d:%d", k.Config.LocalPort, k.Config.RemotePort))
	if k.Config.LocalHost != "" && k.Config.LocalHost != "127.0.0.1" {
		args = append(args, "--address", k.Config.LocalHost)
	}

	k.cmd = exec.CommandContext(ctx, "kubectl", args...)

	// Capture stderr for error reporting.
	stderr, _ := k.cmd.StderrPipe()
	stdout, _ := k.cmd.StdoutPipe()

	log.Info().
		Str("tunnel", k.Config.Name).
		Strs("args", args).
		Msg("k8s: starting kubectl port-forward")

	if err := k.cmd.Start(); err != nil {
		k.alloc.Release(k.Config.LocalPort, k.Config.Name)
		k.setError(fmt.Errorf("kubectl start: %w", err))
		return err
	}

	// Watch stdout for the "Forwarding from" line that indicates success.
	go func() {
		scanner := bufio.NewScanner(stdout)
		for scanner.Scan() {
			line := scanner.Text()
			if strings.Contains(line, "Forwarding from") {
				k.mu.Lock()
				k.status = StatusActive
				k.connectedAt = time.Now()
				k.mu.Unlock()
				k.bus.Publish(event.Event{
					Type:       event.TunnelConnected,
					TunnelName: k.Config.Name,
					Message:    fmt.Sprintf("k8s: %s %s:%d → %s:%d", k.Config.K8sResource, k.Config.LocalHost, k.Config.LocalPort, k.Config.K8sResource, k.Config.RemotePort),
				})
			}
		}
	}()

	// Watch stderr for errors.
	go func() {
		scanner := bufio.NewScanner(stderr)
		var lastErr string
		for scanner.Scan() {
			lastErr = scanner.Text()
			log.Debug().Str("tunnel", k.Config.Name).Str("stderr", lastErr).Msg("k8s: stderr")
		}
		// Process exited — check if it was a clean shutdown.
		if err := k.cmd.Wait(); err != nil {
			select {
			case <-ctx.Done():
				return // Normal disconnect.
			default:
			}
			if lastErr == "" {
				lastErr = err.Error()
			}
			k.setError(fmt.Errorf("kubectl exited: %s", lastErr))
		}
		k.alloc.Release(k.Config.LocalPort, k.Config.Name)
	}()

	// Probe the local port to confirm it's listening.
	go func() {
		for i := 0; i < 20; i++ {
			time.Sleep(250 * time.Millisecond)
			conn, err := net.DialTimeout("tcp", fmt.Sprintf("%s:%d", k.Config.LocalHost, k.Config.LocalPort), 500*time.Millisecond)
			if err == nil {
				conn.Close()
				k.mu.Lock()
				if k.status == StatusConnecting {
					k.status = StatusActive
					k.connectedAt = time.Now()
				}
				k.mu.Unlock()
				return
			}
		}
	}()

	return nil
}

// Disconnect stops the kubectl process.
func (k *K8sTunnel) Disconnect() {
	k.mu.Lock()
	if k.status == StatusStopped {
		k.mu.Unlock()
		return
	}
	cancel := k.cancel
	cmd := k.cmd
	k.status = StatusStopped
	k.mu.Unlock()

	log.Info().Str("tunnel", k.Config.Name).Msg("k8s: disconnecting")

	if cancel != nil {
		cancel()
	}
	if cmd != nil && cmd.Process != nil {
		cmd.Process.Kill()
	}

	k.alloc.Release(k.Config.LocalPort, k.Config.Name)

	k.bus.Publish(event.Event{
		Type:       event.TunnelDisconnected,
		TunnelName: k.Config.Name,
	})
}

func (k *K8sTunnel) setError(err error) {
	k.mu.Lock()
	k.status = StatusError
	k.errorMessage = err.Error()
	k.lastErrorAt = time.Now()
	k.mu.Unlock()

	log.Error().Err(err).Str("tunnel", k.Config.Name).Msg("k8s: tunnel error")
	k.bus.Publish(event.Event{
		Type:       event.TunnelError,
		TunnelName: k.Config.Name,
		Message:    err.Error(),
	})
}
