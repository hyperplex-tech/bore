//go:build darwin

package auth

import (
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"golang.org/x/crypto/ssh/agent"
)

// dialAgent connects to the SSH agent on macOS.
// GUI apps and launchd services don't inherit SSH_AUTH_SOCK, so we try
// multiple discovery methods to find the agent socket.
func dialAgent() (net.Conn, error) {
	candidates := findAgentSockets()
	if len(candidates) == 0 {
		return nil, fmt.Errorf("SSH agent not found (checked SSH_AUTH_SOCK, launchctl, and /private/tmp)")
	}

	// Try each candidate socket. Prefer one that actually has keys loaded,
	// since a system may have multiple agents (e.g. gpg-agent with no keys
	// and the macOS system agent with keys).
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
	return nil, fmt.Errorf("SSH agent not found (checked SSH_AUTH_SOCK, launchctl, and /private/tmp)")
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

	// 2. Ask launchd for the env var (works if shell exported it to launchd).
	if out, err := exec.Command("launchctl", "getenv", "SSH_AUTH_SOCK").Output(); err == nil {
		add(string(out))
	}

	// 3. macOS system SSH agent socket: /private/tmp/com.apple.launchd.*/Listeners
	matches, _ := filepath.Glob("/private/tmp/com.apple.launchd.*/Listeners")
	for _, m := range matches {
		add(m)
	}

	return socks
}
