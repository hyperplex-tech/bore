package tui

import (
	"strings"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

type filterInput struct {
	active bool
	input  textinput.Model
	query  string
}

func newFilterInput() filterInput {
	ti := textinput.New()
	ti.Placeholder = "type to filter tunnels..."
	ti.SetWidth(30)
	s := ti.Styles()
	s.Focused.Text = lipgloss.NewStyle().Foreground(lipgloss.Color("#e0e0e0"))
	s.Focused.Placeholder = lipgloss.NewStyle().Foreground(lipgloss.Color("#555555"))
	ti.SetStyles(s)
	return filterInput{input: ti}
}

func (f *filterInput) activate() tea.Cmd {
	f.active = true
	f.input.SetValue("")
	f.query = ""
	return f.input.Focus()
}

func (f *filterInput) deactivate() {
	f.active = false
	f.input.Blur()
	f.query = ""
	f.input.SetValue("")
}

func (f *filterInput) update(msg tea.Msg) (bool, tea.Cmd) {
	if !f.active {
		return false, nil
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "esc":
			f.deactivate()
			return true, nil
		case "enter":
			f.query = f.input.Value()
			f.active = false
			f.input.Blur()
			return true, nil
		}
	}

	var cmd tea.Cmd
	f.input, cmd = f.input.Update(msg)
	f.query = f.input.Value()
	return true, cmd
}

func (f *filterInput) view(s styles) string {
	if !f.active && f.query == "" {
		return ""
	}

	label := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#4a9eff")).
		Bold(true).
		Render("/ ")

	if f.active {
		return label + f.input.View()
	}
	return label + lipgloss.NewStyle().
		Foreground(lipgloss.Color("#e0e0e0")).
		Render(f.query) +
		lipgloss.NewStyle().Foreground(lipgloss.Color("#555555")).
			Render("  (Esc to clear)")
}

// matches returns true if the tunnel name matches the current filter query.
func (f *filterInput) matches(name string) bool {
	if f.query == "" {
		return true
	}
	return strings.Contains(strings.ToLower(name), strings.ToLower(f.query))
}
