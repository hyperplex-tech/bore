package engine

import (
	"fmt"
	"net"
	"time"

	"github.com/rs/zerolog/log"
	"golang.org/x/crypto/ssh"

	"github.com/hyperplex-tech/bore/internal/auth"
)

// dialSSHViaJumpHosts dials the final SSH target through a chain of jump hosts.
// Each hop in the chain is an SSH connection piped through the previous one.
// Returns the final *ssh.Client; the caller is responsible for closing it
// (which cascades to close all intermediate connections).
func dialSSHViaJumpHosts(jumpHosts []string, target string, sshConfig *ssh.ClientConfig) (*ssh.Client, error) {
	if len(jumpHosts) == 0 {
		return ssh.Dial("tcp", target, sshConfig)
	}

	var current *ssh.Client

	// Dial each jump host in sequence.
	for i, hop := range jumpHosts {
		hopAddr := normalizeAddr(hop)
		hopUser, hopHost := parseUserHost(hopAddr, sshConfig.User)

		hopConfig := &ssh.ClientConfig{
			User:            hopUser,
			Auth:            resolveJumpAuth(hopUser),
			HostKeyCallback: ssh.InsecureIgnoreHostKey(), // TODO: proper host key verification
			Timeout:         15 * time.Second,
		}

		if current == nil {
			// First hop — dial directly.
			log.Debug().Str("hop", hopAddr).Int("step", i+1).Msg("jump: dialing first hop")
			client, err := ssh.Dial("tcp", hopHost, hopConfig)
			if err != nil {
				return nil, fmt.Errorf("jump hop %d (%s): %w", i+1, hopAddr, err)
			}
			current = client
		} else {
			// Subsequent hops — dial through the current connection.
			log.Debug().Str("hop", hopAddr).Int("step", i+1).Msg("jump: dialing through previous hop")
			conn, err := current.Dial("tcp", hopHost)
			if err != nil {
				current.Close()
				return nil, fmt.Errorf("jump hop %d (%s): %w", i+1, hopAddr, err)
			}
			ncc, chans, reqs, err := ssh.NewClientConn(conn, hopHost, hopConfig)
			if err != nil {
				conn.Close()
				current.Close()
				return nil, fmt.Errorf("jump hop %d SSH handshake (%s): %w", i+1, hopAddr, err)
			}
			current = ssh.NewClient(ncc, chans, reqs)
		}
	}

	// Final hop: dial the actual target through the last jump host.
	log.Debug().Str("target", target).Msg("jump: dialing final target")
	conn, err := current.Dial("tcp", target)
	if err != nil {
		current.Close()
		return nil, fmt.Errorf("jump final target (%s): %w", target, err)
	}
	ncc, chans, reqs, err := ssh.NewClientConn(conn, target, sshConfig)
	if err != nil {
		conn.Close()
		current.Close()
		return nil, fmt.Errorf("jump final SSH handshake (%s): %w", target, err)
	}

	return ssh.NewClient(ncc, chans, reqs), nil
}

// normalizeAddr ensures an address has a port (defaults to :22).
func normalizeAddr(addr string) string {
	_, _, err := net.SplitHostPort(addr)
	if err != nil {
		// No port specified — add default SSH port.
		return addr + ":22"
	}
	return addr
}

// parseUserHost splits "user@host:port" into (user, "host:port").
// If no user is specified, defaults to defaultUser.
func parseUserHost(addr, defaultUser string) (string, string) {
	host, port, _ := net.SplitHostPort(addr)

	// Check for user@host.
	user := defaultUser
	for i, c := range host {
		if c == '@' {
			user = host[:i]
			host = host[i+1:]
			break
		}
	}

	return user, net.JoinHostPort(host, port)
}

// resolveJumpAuth resolves auth methods for a jump host.
// Jump hosts use the default composite provider (agent + key auto-discovery).
func resolveJumpAuth(user string) []ssh.AuthMethod {
	provider := auth.NewProvider("")
	methods, err := provider.AuthMethods(auth.AuthConfig{
		SSHUser: user,
	})
	if err != nil {
		return nil
	}
	return methods
}
