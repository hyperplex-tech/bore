package service

import (
	"context"
	"os"

	borev1 "github.com/hyperplex-tech/bore/gen/bore/v1"
	"github.com/hyperplex-tech/bore/internal/auth"
	"github.com/hyperplex-tech/bore/internal/engine"
	"github.com/hyperplex-tech/bore/internal/store"
	"github.com/hyperplex-tech/bore/internal/tailscale"
	"github.com/hyperplex-tech/bore/internal/version"
	"golang.org/x/crypto/ssh/agent"
)

// DaemonService implements the DaemonService gRPC service.
type DaemonService struct {
	borev1.UnimplementedDaemonServiceServer

	store        *store.Store
	engine       *engine.Engine
	configPath   string
	socketPath   string
	reloadConfig func() error
}

// NewDaemonService creates a new DaemonService.
func NewDaemonService(st *store.Store, eng *engine.Engine, configPath, socketPath string, reloadFn func() error) *DaemonService {
	return &DaemonService{
		store:        st,
		engine:       eng,
		configPath:   configPath,
		socketPath:   socketPath,
		reloadConfig: reloadFn,
	}
}

func (s *DaemonService) Status(ctx context.Context, req *borev1.StatusRequest) (*borev1.StatusResponse, error) {
	total := s.engine.TotalCount()
	active := s.engine.ActiveCount()

	agentAvailable, agentKeys := checkSSHAgent()
	ts := tailscale.Detect()

	return &borev1.StatusResponse{
		Version:            version.Version,
		ActiveTunnels:      int32(active),
		TotalTunnels:       int32(total),
		SocketPath:         s.socketPath,
		ConfigPath:         s.configPath,
		SshAgentAvailable:  agentAvailable,
		SshAgentKeys:       int32(agentKeys),
		TailscaleAvailable: ts.Available,
		TailscaleConnected: ts.Running,
		TailscaleIp:        ts.IP,
		TailscaleHostname:  ts.Hostname,
	}, nil
}

func (s *DaemonService) Shutdown(ctx context.Context, req *borev1.ShutdownRequest) (*borev1.ShutdownResponse, error) {
	// Send SIGTERM to ourselves to trigger graceful shutdown.
	p, _ := os.FindProcess(os.Getpid())
	p.Signal(os.Interrupt)
	return &borev1.ShutdownResponse{}, nil
}

func (s *DaemonService) ReloadConfig(ctx context.Context, req *borev1.ReloadConfigRequest) (*borev1.ReloadConfigResponse, error) {
	if err := s.reloadConfig(); err != nil {
		return nil, err
	}
	return &borev1.ReloadConfigResponse{
		TunnelsLoaded: int32(s.engine.TotalCount()),
	}, nil
}

func checkSSHAgent() (available bool, numKeys int) {
	conn, err := auth.DialAgent()
	if err != nil {
		return false, 0
	}
	defer conn.Close()

	ag := agent.NewClient(conn)
	keys, err := ag.List()
	if err != nil {
		return true, 0
	}
	return true, len(keys)
}
