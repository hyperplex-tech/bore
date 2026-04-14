//go:build !windows && !darwin

package auth

import (
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"golang.org/x/crypto/ssh/agent"
)

// dialAgent connects to the SSH agent on Linux.
// Systemd services and GUI apps don't always inherit SSH_AUTH_SOCK, so we
// try multiple discovery methods to find the agent socket.
func dialAgent() (net.Conn, error) {
	candidates := findAgentSockets()
	if len(candidates) == 0 {
		return nil, fmt.Errorf("SSH agent not found (checked SSH_AUTH_SOCK, XDG_RUNTIME_DIR, and /tmp)")
	}

	// Try each candidate socket. Prefer one that actually has keys loaded,
	// since a system may have multiple agents (e.g. gpg-agent with no keys
	// and gnome-keyring with keys loaded).
	var firstConn net.Conn
	for _, sock := range candidates {
		conn, err := net.Dial("unix", sock)
		if err != nil {
			continue
		}
		// Check if this agent has keys.
		ag := agent.NewClient(conn)
		keys, err := ag.List()
		if err == nil && len(keys) > 0 {
			return conn, nil
		}
		// Connectable but no keys — keep as fallback.
		if firstConn == nil {
			firstConn = conn
		} else {
			conn.Close()
		}
	}
	if firstConn != nil {
		return firstConn, nil
	}
	return nil, fmt.Errorf("SSH agent not found (checked SSH_AUTH_SOCK, XDG_RUNTIME_DIR, and /tmp)")
}

// findAgentSockets returns candidate SSH agent socket paths in priority order.
func findAgentSockets() []string {
	var socks []string
	seen := map[string]bool{}

	add := func(s string) {
		s = strings.TrimSpace(s)
		if s != "" && !seen[s] {
			seen[s] = true
			socks = append(socks, s)
		}
	}

	// 1. Standard env var (set in terminal sessions).
	add(os.Getenv("SSH_AUTH_SOCK"))

	// 2. XDG_RUNTIME_DIR-based sockets (systemd user services).
	runtimeDir := os.Getenv("XDG_RUNTIME_DIR")
	if runtimeDir == "" {
		runtimeDir = "/run/user/" + strconv.Itoa(os.Getuid())
	}

	// GNOME Keyring SSH agent.
	add(filepath.Join(runtimeDir, "keyring", "ssh"))

	// systemd ssh-agent socket (common user unit setup).
	add(filepath.Join(runtimeDir, "ssh-agent.socket"))

	// gcr/gnome-keyring-daemon socket (newer GNOME).
	add(filepath.Join(runtimeDir, "gcr", "ssh"))

	// 3. /tmp/ssh-*/agent.* (ssh-agent started by the session).
	matches, _ := filepath.Glob("/tmp/ssh-*/agent.*")
	for _, m := range matches {
		add(m)
	}

	return socks
}
