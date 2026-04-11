//go:build windows

package auth

import (
	"fmt"
	"net"
	"os"

	"github.com/Microsoft/go-winio"
)

const windowsSSHAgentPipe = `\\.\pipe\openssh-ssh-agent`

// dialAgent connects to the SSH agent.
// On Windows, tries the OpenSSH named pipe first, then falls back to SSH_AUTH_SOCK.
func dialAgent() (net.Conn, error) {
	// Try the Windows OpenSSH agent named pipe first.
	conn, err := winio.DialPipe(windowsSSHAgentPipe, nil)
	if err == nil {
		return conn, nil
	}

	// Fall back to SSH_AUTH_SOCK (e.g. WSL interop, pageant).
	sock := os.Getenv("SSH_AUTH_SOCK")
	if sock == "" {
		return nil, fmt.Errorf("SSH agent not available (tried %s and SSH_AUTH_SOCK)", windowsSSHAgentPipe)
	}
	conn2, err := net.Dial("unix", sock)
	if err != nil {
		return nil, fmt.Errorf("connecting to SSH agent: %w", err)
	}
	return conn2, nil
}
