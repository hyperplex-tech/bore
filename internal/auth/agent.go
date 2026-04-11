package auth

import (
	"net"

	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/agent"
)

// DialAgent connects to the platform's SSH agent and returns the connection.
func DialAgent() (net.Conn, error) {
	return dialAgent()
}

// AgentProvider authenticates via the SSH agent (SSH_AUTH_SOCK).
type AgentProvider struct{}

func (p *AgentProvider) AuthMethods(cfg AuthConfig) ([]ssh.AuthMethod, error) {
	conn, err := dialAgent()
	if err != nil {
		return nil, err
	}
	// Note: we don't close conn here — the SSH client will use the agent
	// throughout the connection's lifetime. The conn gets cleaned up when
	// the process exits or the SSH client is closed.

	ag := agent.NewClient(conn)
	return []ssh.AuthMethod{ssh.PublicKeysCallback(ag.Signers)}, nil
}
