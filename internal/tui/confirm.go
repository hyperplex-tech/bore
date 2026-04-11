package tui

import (
	"fmt"

	"charm.land/lipgloss/v2"
)

type confirmAction int

const (
	confirmNone confirmAction = iota
	confirmDeleteTunnel
	confirmDeleteGroup
)

type confirmDialog struct {
	visible bool
	action  confirmAction
	target  string // name of tunnel or group
	width   int
}

func (cd *confirmDialog) show(action confirmAction, target string) {
	cd.visible = true
	cd.action = action
	cd.target = target
}

func (cd *confirmDialog) hide() {
	cd.visible = false
	cd.action = confirmNone
	cd.target = ""
}

func (cd *confirmDialog) view(s styles) string {
	if !cd.visible {
		return ""
	}

	var title, message string
	switch cd.action {
	case confirmDeleteTunnel:
		title = "Delete Tunnel"
		message = fmt.Sprintf("Are you sure you want to delete %q?\nThis cannot be undone.", cd.target)
	case confirmDeleteGroup:
		title = "Delete Group"
		message = fmt.Sprintf("Are you sure you want to delete group %q?\nGroup must be empty.", cd.target)
	default:
		return ""
	}

	titleStyle := lipgloss.NewStyle().Bold(true).
		Foreground(lipgloss.Color("#ef4444"))

	hint := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#888888")).
		Render("\ny = confirm  n/Esc = cancel")

	content := titleStyle.Render(title) + "\n\n" + message + hint

	boxStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#ef4444")).
		Padding(1, 2).
		Width(min(50, cd.width-4)).
		Background(lipgloss.Color("#1a1a2e"))

	return boxStyle.Render(content)
}
