package service

import (
	"context"
	"time"

	"google.golang.org/protobuf/types/known/timestamppb"

	borev1 "github.com/hyperplex-tech/bore/gen/bore/v1"
	"github.com/hyperplex-tech/bore/internal/config"
	"github.com/hyperplex-tech/bore/internal/engine"
	"github.com/hyperplex-tech/bore/internal/event"
	"github.com/hyperplex-tech/bore/internal/store"
)

// TunnelService implements the TunnelService gRPC service.
type TunnelService struct {
	borev1.UnimplementedTunnelServiceServer

	getConfig func() *config.Config
	engine    *engine.Engine
	bus       *event.Bus
	store     *store.Store
}

// NewTunnelService creates a new TunnelService.
func NewTunnelService(getConfig func() *config.Config, eng *engine.Engine, bus *event.Bus, st *store.Store) *TunnelService {
	return &TunnelService{getConfig: getConfig, engine: eng, bus: bus, store: st}
}

func (s *TunnelService) List(ctx context.Context, req *borev1.ListRequest) (*borev1.ListResponse, error) {
	infos := s.engine.List(req.Group)

	var tunnels []*borev1.Tunnel
	for _, info := range infos {
		t := infoToProto(info)
		if req.StatusFilter != borev1.TunnelStatus_TUNNEL_STATUS_UNSPECIFIED && t.Status != req.StatusFilter {
			continue
		}
		tunnels = append(tunnels, t)
	}

	return &borev1.ListResponse{Tunnels: tunnels}, nil
}

func (s *TunnelService) Connect(ctx context.Context, req *borev1.ConnectRequest) (*borev1.ConnectResponse, error) {
	infos, err := s.engine.Connect(req.Names, req.Group, s.getConfig())
	if err != nil {
		return nil, err
	}

	var tunnels []*borev1.Tunnel
	for _, info := range infos {
		tunnels = append(tunnels, infoToProto(info))
	}
	return &borev1.ConnectResponse{Tunnels: tunnels}, nil
}

func (s *TunnelService) Disconnect(ctx context.Context, req *borev1.DisconnectRequest) (*borev1.DisconnectResponse, error) {
	infos, err := s.engine.Disconnect(req.Names, req.Group, s.getConfig())
	if err != nil {
		return nil, err
	}

	var tunnels []*borev1.Tunnel
	for _, info := range infos {
		tunnels = append(tunnels, infoToProto(info))
	}
	return &borev1.DisconnectResponse{Tunnels: tunnels}, nil
}

func (s *TunnelService) Pause(ctx context.Context, req *borev1.PauseRequest) (*borev1.PauseResponse, error) {
	return &borev1.PauseResponse{}, nil
}

func (s *TunnelService) Retry(ctx context.Context, req *borev1.RetryRequest) (*borev1.RetryResponse, error) {
	return &borev1.RetryResponse{}, nil
}

func (s *TunnelService) GetLogs(req *borev1.GetLogsRequest, stream borev1.TunnelService_GetLogsServer) error {
	limit := int(req.Tail)
	if limit <= 0 {
		limit = 100
	}

	// Send historical logs.
	entries, err := s.store.GetLogs(req.Name, limit)
	if err != nil {
		return err
	}
	for _, e := range entries {
		ts, _ := time.Parse("2006-01-02 15:04:05", e.Timestamp)
		if err := stream.Send(&borev1.LogEntry{
			Timestamp:  timestamppb.New(ts),
			Level:      e.Level,
			Message:    e.Message,
			TunnelName: e.TunnelName,
		}); err != nil {
			return err
		}
	}

	// If not following, we're done.
	if !req.Follow {
		return nil
	}

	// Stream live events.
	subID, ch := s.bus.Subscribe(64)
	defer s.bus.Unsubscribe(subID)

	for {
		select {
		case <-stream.Context().Done():
			return nil
		case evt, ok := <-ch:
			if !ok {
				return nil
			}
			if req.Name != "" && evt.TunnelName != req.Name {
				continue
			}
			if err := stream.Send(&borev1.LogEntry{
				Timestamp:  timestamppb.New(evt.Timestamp),
				Level:      eventTypeToLevel(evt.Type),
				Message:    evt.Message,
				TunnelName: evt.TunnelName,
			}); err != nil {
				return err
			}
		}
	}
}

func eventTypeToLevel(t event.Type) string {
	switch t {
	case event.TunnelError:
		return "error"
	case event.TunnelRetrying:
		return "warn"
	default:
		return "info"
	}
}

func infoToProto(info engine.TunnelInfo) *borev1.Tunnel {
	t := &borev1.Tunnel{
		Name:         info.Name,
		Group:        info.Group,
		Status:       statusToProto(info.Status),
		LocalPort:    int32(info.LocalPort),
		LocalHost:    info.LocalHost,
		RemoteHost:   info.RemoteHost,
		RemotePort:   int32(info.RemotePort),
		SshHost:      info.SSHHost,
		SshPort:      int32(info.SSHPort),
		SshUser:      info.SSHUser,
		AuthMethod:   info.Config.AuthMethod,
		IdentityFile: info.Config.IdentityFile,
		JumpHosts:    info.Config.JumpHosts,
		ErrorMessage: info.ErrorMessage,
		RetryCount:    int32(info.RetryCount),
		NextRetrySecs: int32(info.NextRetrySecs),
		K8SContext:   info.Config.K8sContext,
		K8SNamespace: info.Config.K8sNamespace,
		K8SResource:  info.Config.K8sResource,
	}

	tunnelType := borev1.TunnelType_TUNNEL_TYPE_LOCAL
	switch info.Config.Type {
	case "remote":
		tunnelType = borev1.TunnelType_TUNNEL_TYPE_REMOTE
	case "dynamic":
		tunnelType = borev1.TunnelType_TUNNEL_TYPE_DYNAMIC
	case "k8s":
		tunnelType = borev1.TunnelType_TUNNEL_TYPE_K8S
	}
	t.Type = tunnelType

	if !info.ConnectedAt.IsZero() {
		t.ConnectedAt = timestamppb.New(info.ConnectedAt)
	}
	if !info.LastErrorAt.IsZero() {
		t.LastErrorAt = timestamppb.New(info.LastErrorAt)
	}

	return t
}

func statusToProto(s engine.TunnelStatus) borev1.TunnelStatus {
	switch s {
	case engine.StatusActive:
		return borev1.TunnelStatus_TUNNEL_STATUS_ACTIVE
	case engine.StatusConnecting:
		return borev1.TunnelStatus_TUNNEL_STATUS_CONNECTING
	case engine.StatusError:
		return borev1.TunnelStatus_TUNNEL_STATUS_ERROR
	case engine.StatusPaused:
		return borev1.TunnelStatus_TUNNEL_STATUS_PAUSED
	default:
		return borev1.TunnelStatus_TUNNEL_STATUS_STOPPED
	}
}
