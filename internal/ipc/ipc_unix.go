//go:build !windows

package ipc

import (
	"net"
	"os"

	"google.golang.org/grpc"
)

// Listen creates a Unix domain socket listener.
func Listen(addr string) (net.Listener, error) {
	lis, err := net.Listen("unix", addr)
	if err != nil {
		return nil, err
	}
	os.Chmod(addr, 0o600)
	return lis, nil
}

// DialTarget returns the gRPC dial target for a Unix socket.
func DialTarget(addr string) string {
	return "unix://" + addr
}

// DialOption returns a no-op gRPC dial option on Unix (unix:// is natively supported).
func DialOption() grpc.DialOption {
	return grpc.EmptyDialOption{}
}

// Cleanup removes a stale socket file.
func Cleanup(addr string) {
	os.Remove(addr)
}
