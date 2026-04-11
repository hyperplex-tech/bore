package tui

import (
	"fmt"
	"strings"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	internalconfig "github.com/hyperplex-tech/bore/internal/config"
	"github.com/hyperplex-tech/bore/internal/profile"
)

type sshImport struct {
	visible  bool
	tunnels  []internalconfig.TunnelConfig
	selected []bool
	cursor   int
	group    textinput.Model
	focusGrp bool
	width    int
	height   int
	err      string
	loading  bool
}

func newSSHImport() sshImport {
	gi := textinput.New()
	gi.Placeholder = "imported"
	gi.SetWidth(20)
	s := gi.Styles()
	s.Focused.Text = lipgloss.NewStyle().Foreground(lipgloss.Color("#e0e0e0"))
	s.Focused.Placeholder = lipgloss.NewStyle().Foreground(lipgloss.Color("#555555"))
	gi.SetStyles(s)
	return sshImport{group: gi}
}

type sshImportScannedMsg struct {
	tunnels []internalconfig.TunnelConfig
	err     error
}

func scanSSHConfig() tea.Cmd {
	return func() tea.Msg {
		hosts, err := profile.ImportSSHConfig("")
		if err != nil {
			return sshImportScannedMsg{err: err}
		}
		tunnels := profile.ToTunnelConfigs(hosts)
		return sshImportScannedMsg{tunnels: tunnels}
	}
}

func (si *sshImport) show() tea.Cmd {
	si.visible = true
	si.loading = true
	si.tunnels = nil
	si.selected = nil
	si.cursor = 0
	si.focusGrp = false
	si.err = ""
	si.group.SetValue("")
	si.group.Blur()
	return scanSSHConfig()
}

func (si *sshImport) hide() {
	si.visible = false
	si.loading = false
	si.tunnels = nil
	si.selected = nil
	si.group.Blur()
}

func (si *sshImport) onScanned(msg sshImportScannedMsg) {
	si.loading = false
	if msg.err != nil {
		si.err = fmt.Sprintf("Failed to scan: %v", msg.err)
		return
	}
	si.tunnels = msg.tunnels
	si.selected = make([]bool, len(msg.tunnels))
	for i := range si.selected {
		si.selected[i] = true
	}
}

func (si *sshImport) update(msg tea.Msg) (bool, tea.Cmd) {
	if !si.visible {
		return false, nil
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		if si.focusGrp {
			switch msg.String() {
			case "esc":
				si.focusGrp = false
				si.group.Blur()
				return true, nil
			case "enter", "tab":
				si.focusGrp = false
				si.group.Blur()
				return true, nil
			}
			var cmd tea.Cmd
			si.group, cmd = si.group.Update(msg)
			return true, cmd
		}

		switch msg.String() {
		case "esc":
			si.hide()
			return true, nil
		case "j", "down":
			if si.cursor < len(si.tunnels)-1 {
				si.cursor++
			}
			return true, nil
		case "k", "up":
			if si.cursor > 0 {
				si.cursor--
			}
			return true, nil
		case " ":
			if si.cursor < len(si.selected) {
				si.selected[si.cursor] = !si.selected[si.cursor]
			}
			return true, nil
		case "a":
			// Select all
			for i := range si.selected {
				si.selected[i] = true
			}
			return true, nil
		case "n":
			// Deselect all
			for i := range si.selected {
				si.selected[i] = false
			}
			return true, nil
		case "g":
			si.focusGrp = true
			return true, si.group.Focus()
		case "ctrl+s", "enter":
			// Submit handled by caller
			return true, nil
		}
	}

	return true, nil
}

func (si *sshImport) selectedTunnels() []internalconfig.TunnelConfig {
	var result []internalconfig.TunnelConfig
	for i, tc := range si.tunnels {
		if i < len(si.selected) && si.selected[i] {
			result = append(result, tc)
		}
	}
	return result
}

func (si *sshImport) groupName() string {
	g := strings.TrimSpace(si.group.Value())
	if g == "" {
		return "imported"
	}
	return g
}

func (si *sshImport) view(s styles) string {
	if !si.visible {
		return ""
	}

	titleStyle := lipgloss.NewStyle().Bold(true).
		Foreground(lipgloss.Color("#4a9eff"))

	var b strings.Builder
	b.WriteString(titleStyle.Render("Import from SSH Config"))
	b.WriteString("\n\n")

	if si.loading {
		b.WriteString(s.tunnelDim.Render("Scanning ~/.ssh/config..."))
	} else if si.err != "" {
		errStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#ef4444"))
		b.WriteString(errStyle.Render(si.err))
	} else if len(si.tunnels) == 0 {
		b.WriteString(s.tunnelDim.Render("No hosts with LocalForward found."))
	} else {
		// Group input
		groupLabel := lipgloss.NewStyle().Foreground(lipgloss.Color("#888888"))
		b.WriteString(groupLabel.Render("Import to group: "))
		if si.focusGrp {
			b.WriteString(si.group.View())
		} else {
			gn := si.groupName()
			b.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("#e0e0e0")).Render(gn))
			b.WriteString(s.tunnelDim.Render("  (g to edit)"))
		}
		b.WriteString("\n\n")

		// Count selected
		selCount := 0
		for _, sel := range si.selected {
			if sel {
				selCount++
			}
		}
		b.WriteString(s.tunnelDim.Render(fmt.Sprintf(
			"%d/%d selected  (Space toggle  a all  n none)",
			selCount, len(si.tunnels))))
		b.WriteString("\n\n")

		// List
		maxVisible := si.height - 12
		if maxVisible < 3 {
			maxVisible = 3
		}
		start := 0
		if si.cursor >= maxVisible {
			start = si.cursor - maxVisible + 1
		}
		end := start + maxVisible
		if end > len(si.tunnels) {
			end = len(si.tunnels)
		}

		for i := start; i < end; i++ {
			tc := si.tunnels[i]
			check := "[ ]"
			if si.selected[i] {
				check = "[x]"
			}

			cursor := "  "
			if i == si.cursor {
				cursor = "> "
			}

			via := tc.SSHHost
			if tc.SSHUser != "" {
				via = tc.SSHUser + "@" + via
			}

			line := fmt.Sprintf("%s%s %s  localhost:%d → %s:%d via %s",
				cursor, check, tc.Name, tc.LocalPort, tc.RemoteHost, tc.RemotePort, via)
			if i == si.cursor {
				b.WriteString(lipgloss.NewStyle().Bold(true).
					Foreground(lipgloss.Color("#e0e0e0")).
					Render(truncate(line, si.width-8)))
			} else {
				b.WriteString(s.tunnelDim.Render(truncate(line, si.width-8)))
			}
			b.WriteString("\n")
		}
	}

	hint := lipgloss.NewStyle().Foreground(lipgloss.Color("#555555"))
	b.WriteString("\n" + hint.Render("Ctrl+S/Enter import  Esc cancel"))

	boxWidth := min(75, si.width-4)
	boxStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#4a9eff")).
		Padding(1, 2).
		Width(boxWidth).
		Background(lipgloss.Color("#1a1a2e"))

	return boxStyle.Render(b.String())
}
