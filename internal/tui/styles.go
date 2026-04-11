package tui

import (
	"charm.land/lipgloss/v2"
)

type styles struct {
	titleBar    lipgloss.Style
	sidebar     lipgloss.Style
	mainPanel   lipgloss.Style
	statusBar   lipgloss.Style
	selected    lipgloss.Style
	groupItem   lipgloss.Style
	groupActive lipgloss.Style
	sectionHead lipgloss.Style
	keyHint     lipgloss.Style

	// Status dots
	activeDot     string
	stoppedDot    string
	errorDot      string
	connectingDot string
	retryingDot   string

	// Text
	tunnelName  lipgloss.Style
	tunnelDim   lipgloss.Style
	tunnelError lipgloss.Style
	badge       lipgloss.Style
	actionHint  lipgloss.Style
}

func newStyles() styles {
	active := lipgloss.Color("#22c55e")
	errColor := lipgloss.Color("#ef4444")
	warning := lipgloss.Color("#f59e0b")
	stopped := lipgloss.Color("#6b7280")
	connecting := lipgloss.Color("#3b82f6")
	accent := lipgloss.Color("#4a9eff")
	muted := lipgloss.Color("#888888")
	dim := lipgloss.Color("#555555")
	surface := lipgloss.Color("#232338")
	border := lipgloss.Color("#3a3a50")

	return styles{
		titleBar: lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#e0e0e0")).
			Background(surface).
			Padding(0, 1),

		sidebar: lipgloss.NewStyle().
			BorderStyle(lipgloss.NormalBorder()).
			BorderRight(true).
			BorderForeground(border),

		mainPanel: lipgloss.NewStyle().
			Padding(0, 1),

		statusBar: lipgloss.NewStyle().
			Foreground(muted).
			Background(surface).
			Padding(0, 1),

		selected: lipgloss.NewStyle().
			Background(lipgloss.Color("#2a2a50")).
			Bold(true),

		groupItem: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#e0e0e0")).
			Padding(0, 1),

		groupActive: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#ffffff")).
			Background(accent).
			Padding(0, 1),

		sectionHead: lipgloss.NewStyle().
			Foreground(dim).
			Bold(true).
			Padding(0, 1),

		keyHint: lipgloss.NewStyle().
			Foreground(muted).
			Padding(0, 1),

		activeDot:     lipgloss.NewStyle().Foreground(active).Render("●"),
		stoppedDot:    lipgloss.NewStyle().Foreground(stopped).Render("○"),
		errorDot:      lipgloss.NewStyle().Foreground(errColor).Render("✖"),
		connectingDot: lipgloss.NewStyle().Foreground(connecting).Render("◌"),
		retryingDot:   lipgloss.NewStyle().Foreground(warning).Render("◌"),

		tunnelName: lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#e0e0e0")),

		tunnelDim: lipgloss.NewStyle().
			Foreground(muted),

		tunnelError: lipgloss.NewStyle().
			Foreground(errColor),

		badge: lipgloss.NewStyle().
			Foreground(active).
			Bold(true),

		actionHint: lipgloss.NewStyle().
			Foreground(dim),
	}
}

func statusDot(s styles, status borev1Status) string {
	switch status {
	case "active":
		return s.activeDot
	case "error":
		return s.errorDot
	case "connecting":
		return s.connectingDot
	case "retrying":
		return s.retryingDot
	default:
		return s.stoppedDot
	}
}

// borev1Status is a type alias to avoid importing the proto package in styles.
type borev1Status = string
