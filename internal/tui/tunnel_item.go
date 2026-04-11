package tui

import (
	"fmt"
	"strings"
	"time"

	"charm.land/lipgloss/v2"

	borev1 "github.com/hyperplex-tech/bore/gen/bore/v1"
)

func renderTunnelItem(t *borev1.Tunnel, selected bool, width int, s styles) string {
	status := protoStatus(t.Status)
	dot := statusDot(s, status)

	// Line 1: dot + name + status badge + action hint
	name := s.tunnelName.Render(t.Name)
	statusBadge := renderStatusBadge(t, s)
	hint := renderActionHint(t, s)

	line1Left := fmt.Sprintf(" %s %s  %s", dot, name, statusBadge)
	line1Right := hint
	gap := width - lipgloss.Width(line1Left) - lipgloss.Width(line1Right) - 2
	if gap < 1 {
		gap = 1
	}
	line1 := line1Left + strings.Repeat(" ", gap) + line1Right

	// Line 2: connection details or error
	var line2 string
	if status == "error" && t.ErrorMessage != "" {
		line2 = fmt.Sprintf("   %s", s.tunnelError.Render(truncate(t.ErrorMessage, width-6)))
	} else {
		local := fmt.Sprintf("%s:%d", t.LocalHost, t.LocalPort)
		remote := fmt.Sprintf("%s:%d", t.RemoteHost, t.RemotePort)
		via := t.SshHost
		if via == "" && t.K8SContext != "" {
			via = fmt.Sprintf("k8s:%s/%s", t.K8SNamespace, t.K8SResource)
		}
		uptime := ""
		if status == "active" && t.ConnectedAt != nil {
			uptime = fmt.Sprintf("  %s", formatUptime(t.ConnectedAt.AsTime()))
		}
		detail := fmt.Sprintf("   %s → %s", local, remote)
		if via != "" {
			detail += fmt.Sprintf("  via %s", via)
		}
		detail += uptime
		line2 = s.tunnelDim.Render(truncate(detail, width-2))
	}

	content := line1 + "\n" + line2
	if selected {
		return s.selected.Width(width).Render(content)
	}
	return content
}

func renderStatusBadge(t *borev1.Tunnel, s styles) string {
	status := protoStatus(t.Status)
	switch status {
	case "active":
		return s.badge.Render("active")
	case "error":
		if t.NextRetrySecs > 0 {
			return s.tunnelError.Render(fmt.Sprintf("error — retry %ds", t.NextRetrySecs))
		}
		return s.tunnelError.Render("error")
	case "connecting":
		return s.tunnelDim.Render("connecting...")
	case "retrying":
		return s.tunnelError.Render(fmt.Sprintf("retrying in %ds", t.NextRetrySecs))
	case "paused":
		return s.tunnelDim.Render("paused")
	default:
		return s.tunnelDim.Render("stopped")
	}
}

func renderActionHint(t *borev1.Tunnel, s styles) string {
	status := protoStatus(t.Status)
	switch status {
	case "active":
		return s.actionHint.Render("[d]isconnect")
	case "stopped", "paused":
		return s.actionHint.Render("[c]onnect")
	case "error", "retrying":
		return s.actionHint.Render("[r]etry  [l]ogs")
	default:
		return ""
	}
}

func protoStatus(s borev1.TunnelStatus) string {
	switch s {
	case borev1.TunnelStatus_TUNNEL_STATUS_ACTIVE:
		return "active"
	case borev1.TunnelStatus_TUNNEL_STATUS_CONNECTING:
		return "connecting"
	case borev1.TunnelStatus_TUNNEL_STATUS_ERROR:
		return "error"
	case borev1.TunnelStatus_TUNNEL_STATUS_PAUSED:
		return "paused"
	default:
		return "stopped"
	}
}

func formatUptime(connectedAt time.Time) string {
	d := time.Since(connectedAt)
	if d < time.Minute {
		return fmt.Sprintf("%ds", int(d.Seconds()))
	}
	if d < time.Hour {
		return fmt.Sprintf("%dm", int(d.Minutes()))
	}
	return fmt.Sprintf("%dh%dm", int(d.Hours()), int(d.Minutes())%60)
}

func truncate(s string, maxLen int) string {
	if maxLen < 4 {
		maxLen = 4
	}
	if lipgloss.Width(s) <= maxLen {
		return s
	}
	// Crude truncation — works for ASCII.
	if len(s) > maxLen-1 {
		return s[:maxLen-1] + "…"
	}
	return s
}
