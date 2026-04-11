package cli

import (
	"fmt"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	borev1 "github.com/hyperplex-tech/bore/gen/bore/v1"
	"github.com/hyperplex-tech/bore/internal/config"
	"github.com/hyperplex-tech/bore/internal/ipc"
)

// Clients holds gRPC client stubs for all bore services.
type Clients struct {
	Daemon borev1.DaemonServiceClient
	Tunnel borev1.TunnelServiceClient
	Group  borev1.GroupServiceClient
	Event  borev1.EventServiceClient
	conn   *grpc.ClientConn
}

// Dial connects to the bore daemon over the Unix socket.
func Dial(socketPath string) (*Clients, error) {
	if socketPath == "" {
		socketPath = config.SocketPath()
	}

	conn, err := grpc.NewClient(
		ipc.DialTarget(socketPath),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		ipc.DialOption(),
	)
	if err != nil {
		return nil, fmt.Errorf("connecting to daemon: %w", err)
	}

	return &Clients{
		Daemon: borev1.NewDaemonServiceClient(conn),
		Tunnel: borev1.NewTunnelServiceClient(conn),
		Group:  borev1.NewGroupServiceClient(conn),
		Event:  borev1.NewEventServiceClient(conn),
		conn:   conn,
	}, nil
}

// Close closes the gRPC connection.
func (c *Clients) Close() error {
	return c.conn.Close()
}
