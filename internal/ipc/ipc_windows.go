//go:build windows

package ipc

import (
	"context"
	"net"

	"github.com/Microsoft/go-winio"
	"google.golang.org/grpc"
)

// Listen creates a Windows named pipe listener.
func Listen(addr string) (net.Listener, error) {
	return winio.ListenPipe(addr, &winio.PipeConfig{
		SecurityDescriptor: "D:P(A;;GA;;;OW)", // owner-only access
	})
}

// DialTarget returns a passthrough gRPC target for the named pipe address.
// The actual dialing is handled by DialOption().
func DialTarget(addr string) string {
	return "passthrough:///" + addr
}

// DialOption returns a gRPC dial option that connects via Windows named pipes.
func DialOption() grpc.DialOption {
	return grpc.WithContextDialer(func(ctx context.Context, addr string) (net.Conn, error) {
		return winio.DialPipeContext(ctx, addr)
	})
}

// Cleanup is a no-op on Windows; named pipes are cleaned up automatically.
func Cleanup(addr string) {}
