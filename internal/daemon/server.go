package daemon

import (
	"google.golang.org/grpc"

	borev1 "github.com/hyperplex-tech/bore/gen/bore/v1"
	"github.com/hyperplex-tech/bore/internal/config"
	"github.com/hyperplex-tech/bore/internal/daemon/service"
)

// Server wraps a gRPC server with bore services registered.
type Server struct {
	*grpc.Server
}

// NewServer creates a gRPC server with all bore services.
func NewServer(d *Daemon) *Server {
	s := grpc.NewServer()

	getConfig := func() *config.Config { return d.cfg }
	borev1.RegisterDaemonServiceServer(s, service.NewDaemonService(d.store, d.engine, d.configPath, d.socketPath, d.ReloadConfig))
	borev1.RegisterTunnelServiceServer(s, service.NewTunnelService(getConfig, d.engine, d.bus, d.store))
	borev1.RegisterGroupServiceServer(s, service.NewGroupService(getConfig, d.engine))
	borev1.RegisterEventServiceServer(s, service.NewEventService(d.bus))

	return &Server{Server: s}
}
