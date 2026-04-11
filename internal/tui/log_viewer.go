package tui

import (
	"fmt"
	"strings"
	"time"

	"charm.land/lipgloss/v2"

	borev1 "github.com/hyperplex-tech/bore/gen/bore/v1"
)

type logViewer struct {
	visible    bool
	tunnelName string
	entries    []logLine
	width      int
	height     int
	offset     int // scroll offset from bottom
}

type logLine struct {
	timestamp  time.Time
	level      string
	message    string
	tunnelName string
}

func (lv *logViewer) show(name string) {
	lv.visible = true
	lv.tunnelName = name
	lv.entries = nil
	lv.offset = 0
}

func (lv *logViewer) hide() {
	lv.visible = false
	lv.tunnelName = ""
	lv.entries = nil
}

func (lv *logViewer) addEntry(entry *borev1.LogEntry) {
	ts := time.Now()
	if entry.Timestamp != nil {
		ts = entry.Timestamp.AsTime()
	}
	lv.entries = append(lv.entries, logLine{
		timestamp:  ts,
		level:      entry.Level,
		message:    entry.Message,
		tunnelName: entry.TunnelName,
	})
	// Keep max 200 entries.
	if len(lv.entries) > 200 {
		lv.entries = lv.entries[len(lv.entries)-200:]
	}
}

func (lv *logViewer) view(s styles) string {
	if !lv.visible {
		return ""
	}

	border := lipgloss.NewStyle().
		BorderStyle(lipgloss.NormalBorder()).
		BorderTop(true).
		BorderForeground(lipgloss.Color("#3a3a50")).
		Width(lv.width).
		Height(lv.height)

	headerText := "LOGS: all tunnels"
	if lv.tunnelName != "" {
		headerText = fmt.Sprintf("LOGS: %s", lv.tunnelName)
	}
	header := s.sectionHead.Render(headerText)

	contentHeight := lv.height - 2 // border + header
	if contentHeight < 1 {
		contentHeight = 1
	}

	var lines []string
	start := len(lv.entries) - contentHeight
	if start < 0 {
		start = 0
	}
	for i := start; i < len(lv.entries); i++ {
		e := lv.entries[i]
		ts := e.timestamp.Local().Format("15:04:05")
		levelColor := lipgloss.Color("#888888")
		switch e.level {
		case "error":
			levelColor = lipgloss.Color("#ef4444")
		case "warn":
			levelColor = lipgloss.Color("#f59e0b")
		case "info":
			levelColor = lipgloss.Color("#3b82f6")
		}
		lvl := lipgloss.NewStyle().Foreground(levelColor).Render(fmt.Sprintf("[%s]", e.level))
		var line string
		if lv.tunnelName == "" && e.tunnelName != "" {
			tn := lipgloss.NewStyle().Foreground(lipgloss.Color("#a78bfa")).Render(
				fmt.Sprintf("%-16s", truncate(e.tunnelName, 16)))
			line = fmt.Sprintf("%s %s %s %s", s.tunnelDim.Render(ts), lvl, tn, e.message)
		} else {
			line = fmt.Sprintf("%s %s %s", s.tunnelDim.Render(ts), lvl, e.message)
		}
		lines = append(lines, truncate(line, lv.width-2))
	}

	// Pad to fill height.
	for len(lines) < contentHeight {
		lines = append(lines, "")
	}

	content := header + "\n" + strings.Join(lines, "\n")
	return border.Render(content)
}
