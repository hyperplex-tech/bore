package tui

import (
	"fmt"
	"strings"

	borev1 "github.com/hyperplex-tech/bore/gen/bore/v1"
)

type sidebar struct {
	groups  []*borev1.Group
	cursor  int
	focused bool
	width   int
	height  int
}

func newSidebar() sidebar {
	return sidebar{width: 18}
}

func (sb *sidebar) setGroups(groups []*borev1.Group) {
	sb.groups = groups
	if sb.cursor > len(groups) {
		sb.cursor = len(groups)
	}
}

func (sb *sidebar) moveUp() {
	if sb.cursor > 0 {
		sb.cursor--
	}
}

func (sb *sidebar) moveDown() {
	// +1 because index 0 is "All tunnels"
	if sb.cursor < len(sb.groups) {
		sb.cursor++
	}
}

// selectedGroup returns "" for "All tunnels" or the group name.
func (sb *sidebar) selectedGroup() string {
	if sb.cursor == 0 || sb.cursor > len(sb.groups) {
		return ""
	}
	return sb.groups[sb.cursor-1].Name
}

func (sb *sidebar) view(s styles) string {
	var b strings.Builder
	w := sb.width - 2 // padding

	b.WriteString(s.sectionHead.Width(w).Render("GROUPS"))
	b.WriteString("\n")

	// "All tunnels" entry
	label := padRight("All tunnels", w)
	if sb.cursor == 0 {
		if sb.focused {
			b.WriteString(s.groupActive.Width(w).Render(label))
		} else {
			b.WriteString(s.selected.Width(w).Render(label))
		}
	} else {
		b.WriteString(s.groupItem.Width(w).Render(label))
	}
	b.WriteString("\n")

	// Group entries
	for i, g := range sb.groups {
		name := g.Name
		if g.ActiveCount > 0 {
			name = fmt.Sprintf("%s (%d)", g.Name, g.ActiveCount)
		}
		label := padRight(name, w)

		idx := i + 1 // offset by 1 for "All tunnels"
		if idx == sb.cursor {
			if sb.focused {
				b.WriteString(s.groupActive.Width(w).Render(label))
			} else {
				b.WriteString(s.selected.Width(w).Render(label))
			}
		} else {
			b.WriteString(s.groupItem.Width(w).Render(label))
		}
		b.WriteString("\n")
	}

	// Fill remaining space, then key hints
	usedLines := 2 + len(sb.groups) // header + all tunnels + groups
	remaining := sb.height - usedLines - 10
	for i := 0; i < remaining; i++ {
		b.WriteString("\n")
	}

	b.WriteString("\n")
	b.WriteString(s.sectionHead.Width(w).Render("KEYS"))
	b.WriteString("\n")
	b.WriteString(s.keyHint.Render("c connect"))
	b.WriteString("\n")
	b.WriteString(s.keyHint.Render("d disconnect"))
	b.WriteString("\n")
	b.WriteString(s.keyHint.Render("a add  x del"))
	b.WriteString("\n")
	b.WriteString(s.keyHint.Render("e edit l logs"))
	b.WriteString("\n")
	b.WriteString(s.keyHint.Render("C/D all  / flt"))
	b.WriteString("\n")
	b.WriteString(s.keyHint.Render("i import"))
	b.WriteString("\n")
	b.WriteString(s.keyHint.Render("? help q quit"))

	return s.sidebar.Height(sb.height).Width(sb.width).Render(b.String())
}

func padRight(s string, width int) string {
	if len(s) >= width {
		return s[:width]
	}
	return s + strings.Repeat(" ", width-len(s))
}
