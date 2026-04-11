package profile

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/hyperplex-tech/bore/internal/config"
)

// SSHHost represents a parsed SSH config host entry with LocalForward directives.
type SSHHost struct {
	Alias          string
	HostName       string
	User           string
	Port           int
	IdentityFile   string
	IdentitiesOnly bool
	ProxyJump      string
	LocalForwards  []LocalForward
}

// LocalForward is a parsed LocalForward directive.
type LocalForward struct {
	LocalHost  string
	LocalPort  int
	RemoteHost string
	RemotePort int
}

// ImportSSHConfig parses ~/.ssh/config and returns host entries that have
// LocalForward directives (i.e., entries that map to bore tunnels).
// ProxyJump aliases are resolved to real hostnames using the full SSH config.
func ImportSSHConfig(path string) ([]SSHHost, error) {
	if path == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, err
		}
		path = filepath.Join(home, ".ssh", "config")
	}

	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("opening %s: %w", path, err)
	}
	defer f.Close()

	var hosts []SSHHost
	var current *SSHHost

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		key, value := splitKeyValue(line)
		key = strings.ToLower(key)

		switch key {
		case "host":
			// Skip wildcard patterns.
			if strings.ContainsAny(value, "*?") {
				current = nil
				continue
			}
			hosts = append(hosts, SSHHost{Alias: value, Port: 22})
			current = &hosts[len(hosts)-1]
		case "hostname":
			if current != nil {
				current.HostName = value
			}
		case "user":
			if current != nil {
				current.User = value
			}
		case "port":
			if current != nil {
				if p, err := strconv.Atoi(value); err == nil {
					current.Port = p
				}
			}
		case "identityfile":
			if current != nil {
				current.IdentityFile = value
			}
		case "identitiesonly":
			if current != nil {
				current.IdentitiesOnly = strings.EqualFold(value, "yes")
			}
		case "proxyjump":
			if current != nil {
				current.ProxyJump = value
			}
		case "localforward":
			if current != nil {
				lf, err := parseLocalForward(value)
				if err == nil {
					current.LocalForwards = append(current.LocalForwards, lf)
				}
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	// Build alias lookup table for resolving ProxyJump references.
	aliasMap := make(map[string]*SSHHost, len(hosts))
	for i := range hosts {
		aliasMap[hosts[i].Alias] = &hosts[i]
	}

	// Resolve ProxyJump aliases to real hostnames and store resolved metadata.
	for i := range hosts {
		if hosts[i].ProxyJump != "" {
			hosts[i].ProxyJump = resolveProxyJump(hosts[i].ProxyJump, aliasMap)
		}
	}

	// Filter to hosts that have LocalForward directives.
	var result []SSHHost
	for _, h := range hosts {
		if len(h.LocalForwards) > 0 {
			if h.HostName == "" {
				h.HostName = h.Alias
			}
			result = append(result, h)
		}
	}

	return result, nil
}

// resolveProxyJump resolves a comma-separated ProxyJump value, replacing SSH
// config aliases with their real user@hostname:port addresses.
func resolveProxyJump(proxyJump string, aliasMap map[string]*SSHHost) string {
	hops := strings.Split(proxyJump, ",")
	for i, hop := range hops {
		hop = strings.TrimSpace(hop)
		if h, ok := aliasMap[hop]; ok {
			hostname := h.HostName
			if hostname == "" {
				hostname = h.Alias
			}
			resolved := hostname
			if h.User != "" {
				resolved = h.User + "@" + resolved
			}
			if h.Port != 0 && h.Port != 22 {
				resolved = fmt.Sprintf("%s:%d", resolved, h.Port)
			}
			hops[i] = resolved
		}
	}
	return strings.Join(hops, ",")
}

// ToTunnelConfigs converts parsed SSH hosts into bore tunnel configs.
func ToTunnelConfigs(hosts []SSHHost) []config.TunnelConfig {
	var tunnels []config.TunnelConfig
	for _, h := range hosts {
		for i, lf := range h.LocalForwards {
			name := h.Alias
			if len(h.LocalForwards) > 1 {
				name = fmt.Sprintf("%s-%d", h.Alias, i+1)
			}

			tc := config.TunnelConfig{
				Name:         name,
				Type:         "local",
				LocalHost:    lf.LocalHost,
				LocalPort:    lf.LocalPort,
				RemoteHost:   lf.RemoteHost,
				RemotePort:   lf.RemotePort,
				SSHHost:      h.HostName,
				SSHPort:      h.Port,
				SSHUser:      h.User,
				IdentityFile: h.IdentityFile,
			}

			// When IdentitiesOnly is set (or an identity file is specified),
			// use key-only auth to avoid agent key enumeration failures.
			if h.IdentitiesOnly && h.IdentityFile != "" {
				tc.AuthMethod = "key"
			}

			if h.ProxyJump != "" {
				tc.JumpHosts = strings.Split(h.ProxyJump, ",")
			}

			tunnels = append(tunnels, tc)
		}
	}
	return tunnels
}

// splitKeyValue splits "Key Value" or "Key=Value".
func splitKeyValue(line string) (string, string) {
	// Handle "Key=Value".
	if idx := strings.IndexByte(line, '='); idx > 0 {
		before := strings.TrimSpace(line[:idx])
		after := strings.TrimSpace(line[idx+1:])
		if !strings.ContainsAny(before, " \t") {
			return before, after
		}
	}

	// Handle "Key Value" (whitespace separated).
	fields := strings.SplitN(line, " ", 2)
	if len(fields) == 1 {
		fields = strings.SplitN(line, "\t", 2)
	}
	if len(fields) == 2 {
		return strings.TrimSpace(fields[0]), strings.TrimSpace(fields[1])
	}
	return line, ""
}

// parseLocalForward parses "bind_address:port host:hostport" or "port host:hostport".
func parseLocalForward(value string) (LocalForward, error) {
	parts := strings.Fields(value)
	if len(parts) != 2 {
		return LocalForward{}, fmt.Errorf("invalid LocalForward: %s", value)
	}

	local := parts[0]
	remote := parts[1]

	lf := LocalForward{LocalHost: "127.0.0.1"}

	// Parse local side.
	if host, port, err := splitHostPort(local); err == nil {
		lf.LocalHost = host
		lf.LocalPort = port
	} else if p, err := strconv.Atoi(local); err == nil {
		lf.LocalPort = p
	} else {
		return LocalForward{}, fmt.Errorf("invalid local address: %s", local)
	}

	// Parse remote side.
	if host, port, err := splitHostPort(remote); err == nil {
		lf.RemoteHost = host
		lf.RemotePort = port
	} else {
		return LocalForward{}, fmt.Errorf("invalid remote address: %s", remote)
	}

	return lf, nil
}

func splitHostPort(addr string) (string, int, error) {
	lastColon := strings.LastIndex(addr, ":")
	if lastColon < 0 {
		return "", 0, fmt.Errorf("no port in %s", addr)
	}
	host := addr[:lastColon]
	port, err := strconv.Atoi(addr[lastColon+1:])
	if err != nil {
		return "", 0, err
	}
	return host, port, nil
}
