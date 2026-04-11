package tui

import (
	"strings"

	"charm.land/lipgloss/v2"
)

type helpOverlay struct {
	visible bool
	width   int
	height  int
}

func (h *helpOverlay) toggle() {
	h.visible = !h.visible
}

func (h *helpOverlay) view(s styles) string {
	if !h.visible {
		return ""
	}

	title := lipgloss.NewStyle().Bold(true).
		Foreground(lipgloss.Color("#e0e0e0")).
		Render("Bore — Keyboard Shortcuts")

	sections := []struct {
		header string
		keys   [][2]string
	}{
		{"Navigation", [][2]string{
			{"j / ↓", "Move down"},
			{"k / ↑", "Move up"},
			{"Tab", "Switch focus (sidebar / tunnels)"},
			{"/", "Filter tunnels by name"},
			{"Esc", "Clear filter / close dialog"},
		}},
		{"Tunnel Actions", [][2]string{
			{"c", "Connect selected tunnel"},
			{"d", "Disconnect selected tunnel"},
			{"r", "Retry failed tunnel"},
			{"l", "Toggle tunnel log viewer"},
			{"L", "Toggle all logs"},
			{"x", "Delete selected tunnel"},
			{"a", "Add new tunnel"},
			{"e", "Edit selected tunnel"},
			{"p", "Duplicate selected tunnel"},
		}},
		{"Bulk Actions", [][2]string{
			{"C", "Connect all tunnels (or group in sidebar)"},
			{"D", "Disconnect all tunnels (or group in sidebar)"},
		}},
		{"Groups (sidebar focused)", [][2]string{
			{"a", "Add new group"},
			{"r", "Rename group"},
			{"x", "Delete empty group"},
		}},
		{"General", [][2]string{
			{"i", "Import from SSH config"},
			{"?", "Toggle this help"},
			{"q / Ctrl+C", "Quit"},
		}},
	}

	var b strings.Builder
	b.WriteString(title)
	b.WriteString("\n\n")

	for _, sec := range sections {
		header := lipgloss.NewStyle().Bold(true).
			Foreground(lipgloss.Color("#4a9eff")).
			Render(sec.header)
		b.WriteString(header)
		b.WriteString("\n")
		for _, kv := range sec.keys {
			key := lipgloss.NewStyle().
				Foreground(lipgloss.Color("#f59e0b")).
				Width(16).
				Render(kv[0])
			b.WriteString("  " + key + kv[1] + "\n")
		}
		b.WriteString("\n")
	}

	b.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("#888888")).
		Render("Press ? or Esc to close"))

	boxStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#4a9eff")).
		Padding(1, 2).
		Width(min(60, h.width-4)).
		Background(lipgloss.Color("#1a1a2e"))

	return boxStyle.Render(b.String())
}
