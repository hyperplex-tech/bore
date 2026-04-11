package tui

import (
	"strings"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

type groupInput struct {
	visible  bool
	renaming bool   // true when renaming an existing group
	oldName  string // original name when renaming
	input    textinput.Model
	width    int
}

func newGroupInput() groupInput {
	ti := textinput.New()
	ti.Placeholder = "group-name"
	ti.SetWidth(25)
	s := ti.Styles()
	s.Focused.Text = lipgloss.NewStyle().Foreground(lipgloss.Color("#e0e0e0"))
	s.Focused.Placeholder = lipgloss.NewStyle().Foreground(lipgloss.Color("#555555"))
	ti.SetStyles(s)
	return groupInput{input: ti}
}

func (gi *groupInput) show() tea.Cmd {
	gi.visible = true
	gi.renaming = false
	gi.oldName = ""
	gi.input.SetValue("")
	return gi.input.Focus()
}

func (gi *groupInput) showRename(oldName string) tea.Cmd {
	gi.visible = true
	gi.renaming = true
	gi.oldName = oldName
	gi.input.SetValue(oldName)
	return gi.input.Focus()
}

func (gi *groupInput) hide() {
	gi.visible = false
	gi.renaming = false
	gi.oldName = ""
	gi.input.Blur()
	gi.input.SetValue("")
}

func (gi *groupInput) value() string {
	return strings.TrimSpace(gi.input.Value())
}

func (gi *groupInput) update(msg tea.Msg) (bool, tea.Cmd) {
	if !gi.visible {
		return false, nil
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "esc":
			gi.hide()
			return true, nil
		case "enter":
			// Submit handled by caller
			return true, nil
		}
	}

	var cmd tea.Cmd
	gi.input, cmd = gi.input.Update(msg)
	return true, cmd
}

func (gi *groupInput) view(s styles) string {
	if !gi.visible {
		return ""
	}

	titleStyle := lipgloss.NewStyle().Bold(true).
		Foreground(lipgloss.Color("#4a9eff"))

	hint := lipgloss.NewStyle().Foreground(lipgloss.Color("#555555"))

	title := "New Group"
	if gi.renaming {
		title = "Rename Group"
	}

	content := titleStyle.Render(title) + "\n\n" +
		"Name: " + gi.input.View() + "\n\n" +
		hint.Render("Enter confirm  Esc cancel")

	boxStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#4a9eff")).
		Padding(1, 2).
		Width(min(40, gi.width-4)).
		Background(lipgloss.Color("#1a1a2e"))

	return boxStyle.Render(content)
}
