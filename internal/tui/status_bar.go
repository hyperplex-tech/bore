package tui

import (
	"fmt"
	"strings"

	borev1 "github.com/hyperplex-tech/bore/gen/bore/v1"
)

type statusBar struct {
	status           *borev1.StatusResponse
	width            int
	err              error
	autoRefreshCount int
}

func (sb *statusBar) view(s styles) string {
	if sb.status == nil {
		return s.statusBar.Width(sb.width).Render("Connecting to daemon...")
	}

	var parts []string
	parts = append(parts, fmt.Sprintf("bore v%s", sb.status.Version))

	if sb.status.SshAgentAvailable {
		parts = append(parts, fmt.Sprintf("SSH agent: %d keys", sb.status.SshAgentKeys))
	} else {
		parts = append(parts, "SSH agent: N/A")
	}

	if sb.status.TailscaleConnected {
		parts = append(parts, fmt.Sprintf("Tailscale: connected (%s)", sb.status.TailscaleIp))
	} else if sb.status.TailscaleAvailable {
		parts = append(parts, "Tailscale: disconnected")
	}

	if sb.autoRefreshCount > 0 {
		parts = append(parts, fmt.Sprintf("Auto-refresh: %d tunnel(s)", sb.autoRefreshCount))
	}

	path := sb.status.ConfigPath
	if home := findHome(path); home != "" {
		path = "~" + path[len(home):]
	}
	parts = append(parts, fmt.Sprintf("Config: %s", path))

	line := strings.Join(parts, " │ ")
	return s.statusBar.Width(sb.width).Render(line)
}

func findHome(path string) string {
	// Find /home/USER or /Users/USER prefix.
	prefixes := []string{"/home/", "/Users/"}
	for _, p := range prefixes {
		if strings.HasPrefix(path, p) {
			rest := path[len(p):]
			if idx := strings.IndexByte(rest, '/'); idx > 0 {
				return p + rest[:idx]
			}
		}
	}
	return ""
}
