package tui

import (
	"strings"

	borev1 "github.com/hyperplex-tech/bore/gen/bore/v1"
)

type tunnelList struct {
	tunnels    []*borev1.Tunnel
	filtered   []*borev1.Tunnel
	cursor     int
	offset     int // scroll offset
	focused    bool
	width      int
	height     int
	group      string                // current group filter ("" = all)
	textFilter func(name string) bool // optional text filter
}

func newTunnelList() tunnelList {
	return tunnelList{focused: true}
}

func (tl *tunnelList) setTunnels(tunnels []*borev1.Tunnel, group string) {
	tl.tunnels = tunnels
	tl.group = group
	tl.applyFilter()
}

func (tl *tunnelList) applyFilter() {
	tl.filtered = nil
	for _, t := range tl.tunnels {
		if tl.group != "" && t.Group != tl.group {
			continue
		}
		if tl.textFilter != nil && !tl.textFilter(t.Name) {
			continue
		}
		tl.filtered = append(tl.filtered, t)
	}
	// Sort: active first, then error/retrying, then stopped.
	sortTunnels(tl.filtered)

	if tl.cursor >= len(tl.filtered) {
		tl.cursor = max(0, len(tl.filtered)-1)
	}
	tl.ensureVisible()
}

func (tl *tunnelList) moveUp() {
	if tl.cursor > 0 {
		tl.cursor--
		tl.ensureVisible()
	}
}

func (tl *tunnelList) moveDown() {
	if tl.cursor < len(tl.filtered)-1 {
		tl.cursor++
		tl.ensureVisible()
	}
}

func (tl *tunnelList) selectedTunnel() *borev1.Tunnel {
	if tl.cursor >= 0 && tl.cursor < len(tl.filtered) {
		return tl.filtered[tl.cursor]
	}
	return nil
}

func (tl *tunnelList) ensureVisible() {
	itemHeight := 3 // 2 lines per item + 1 gap
	visibleItems := tl.height / itemHeight
	if visibleItems < 1 {
		visibleItems = 1
	}
	if tl.cursor < tl.offset {
		tl.offset = tl.cursor
	}
	if tl.cursor >= tl.offset+visibleItems {
		tl.offset = tl.cursor - visibleItems + 1
	}
}

func (tl *tunnelList) view(s styles, autoRefresh ...map[string]bool) string {
	if len(tl.filtered) == 0 {
		return s.tunnelDim.Render("\n  No tunnels to display")
	}

	itemHeight := 3
	visibleItems := tl.height / itemHeight
	if visibleItems < 1 {
		visibleItems = 1
	}

	var b strings.Builder
	end := tl.offset + visibleItems
	if end > len(tl.filtered) {
		end = len(tl.filtered)
	}

	ar := map[string]bool{}
	if len(autoRefresh) > 0 && autoRefresh[0] != nil {
		ar = autoRefresh[0]
	}
	for i := tl.offset; i < end; i++ {
		selected := i == tl.cursor && tl.focused
		b.WriteString(renderTunnelItem(tl.filtered[i], selected, tl.width, s, ar[tl.filtered[i].Name]))
		b.WriteString("\n")
	}

	return b.String()
}

// sortTunnels sorts by status priority: active, connecting, retrying, error, paused, stopped.
func sortTunnels(tunnels []*borev1.Tunnel) {
	order := map[borev1.TunnelStatus]int{
		borev1.TunnelStatus_TUNNEL_STATUS_ACTIVE:     0,
		borev1.TunnelStatus_TUNNEL_STATUS_CONNECTING: 1,
		borev1.TunnelStatus_TUNNEL_STATUS_ERROR:      2,
		borev1.TunnelStatus_TUNNEL_STATUS_PAUSED:     3,
		borev1.TunnelStatus_TUNNEL_STATUS_STOPPED:    4,
	}
	// Simple insertion sort (lists are small).
	for i := 1; i < len(tunnels); i++ {
		j := i
		for j > 0 && order[tunnels[j].Status] < order[tunnels[j-1].Status] {
			tunnels[j], tunnels[j-1] = tunnels[j-1], tunnels[j]
			j--
		}
	}
}
