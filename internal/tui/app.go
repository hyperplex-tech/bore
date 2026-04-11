package tui

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/bubbles/v2/key"
	"charm.land/lipgloss/v2"

	borev1 "github.com/hyperplex-tech/bore/gen/bore/v1"
	"github.com/hyperplex-tech/bore/internal/cli"
)

type focusArea int

const (
	focusSidebar focusArea = iota
	focusMain
)

// Model is the top-level Bubble Tea model.
type Model struct {
	clients *cli.Clients
	keys    keyMap
	styles  styles

	sidebar    sidebar
	tunnelList tunnelList
	logViewer  logViewer
	statusBar  statusBar

	// Overlays / dialogs
	help      helpOverlay
	confirm   confirmDialog
	form      tunnelForm
	filter    filterInput
	sshImport sshImport
	groupIn   groupInput

	eventStream borev1.EventService_SubscribeClient
	logStream   borev1.TunnelService_GetLogsClient

	// For edit: store the tunnel being edited so we can pass config to the form.
	editingTunnel *borev1.Tunnel

	focus  focusArea
	width  int
	height int
	ready  bool
}

// NewModel creates the initial TUI model.
func NewModel(clients *cli.Clients) Model {
	return Model{
		clients:   clients,
		keys:      defaultKeyMap(),
		styles:    newStyles(),
		sidebar:   newSidebar(),
		tunnelList: newTunnelList(),
		filter:    newFilterInput(),
		form:      newTunnelForm(),
		sshImport: newSSHImport(),
		groupIn:   newGroupInput(),
		focus:     focusMain,
	}
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(
		refreshTunnels(m.clients),
		refreshGroups(m.clients),
		refreshStatus(m.clients),
		subscribeEvents(m.clients),
		tickCmd(),
	)
}


func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.sidebar.height = m.height - 3
		m.sidebar.width = 20
		mainWidth := m.width - m.sidebar.width
		m.tunnelList.width = mainWidth - 2
		logHeight := 0
		if m.logViewer.visible {
			logHeight = max((m.height-3)*30/100, 5)
			m.logViewer.width = mainWidth - 2
			m.logViewer.height = logHeight
		}
		m.tunnelList.height = m.height - 3 - logHeight
		m.statusBar.width = m.width
		m.help.width = m.width
		m.help.height = m.height
		m.confirm.width = m.width
		m.form.width = m.width
		m.form.height = m.height - 4
		m.sshImport.width = m.width
		m.sshImport.height = m.height - 4
		m.groupIn.width = m.width
		m.ready = true
		return m, nil

	case tea.KeyMsg:
		return m.handleKey(msg)

	// --- Data loads ---
	case tunnelsLoadedMsg:
		if msg.err == nil {
			m.tunnelList.setTunnels(msg.tunnels, m.sidebar.selectedGroup())
		}
		return m, nil

	case groupsLoadedMsg:
		if msg.err == nil {
			m.sidebar.setGroups(msg.groups)
		}
		return m, nil

	case daemonStatusMsg:
		if msg.err == nil {
			m.statusBar.status = msg.status
		} else {
			m.statusBar.err = msg.err
		}
		return m, nil

	// --- Event stream ---
	case eventStreamOpenedMsg:
		if msg.err != nil {
			return m, delayedResubscribe(m.clients)
		}
		m.eventStream = msg.stream
		return m, waitForEvent(m.eventStream)

	case tunnelEventMsg:
		if msg.err != nil {
			m.eventStream = nil
			return m, delayedResubscribe(m.clients)
		}
		return m, tea.Batch(
			refreshTunnels(m.clients),
			refreshGroups(m.clients),
			waitForEvent(m.eventStream),
		)

	// --- Log stream ---
	case logStreamOpenedMsg:
		if msg.err != nil {
			return m, nil
		}
		m.logStream = msg.stream
		return m, waitForLogEntry(m.logStream)

	case logEntryMsg:
		if msg.err != nil {
			m.logStream = nil
			return m, nil
		}
		m.logViewer.addEntry(msg.entry)
		return m, waitForLogEntry(m.logStream)

	// --- Action results ---
	case connectResultMsg, disconnectResultMsg, retryResultMsg:
		return m, tea.Batch(refreshTunnels(m.clients), refreshGroups(m.clients))

	// --- Config changes ---
	case configChangedMsg:
		if msg.err != nil {
			if m.form.visible {
				m.form.err = msg.err.Error()
			}
			return m, nil
		}
		m.form.hide()
		m.sshImport.hide()
		m.groupIn.hide()
		m.confirm.hide()
		return m, tea.Batch(
			reloadDaemonConfig(m.clients),
			refreshTunnels(m.clients),
			refreshGroups(m.clients),
		)

	case daemonReloadedMsg:
		return m, tea.Batch(refreshTunnels(m.clients), refreshGroups(m.clients))

	// --- SSH import scan result ---
	case sshImportScannedMsg:
		m.sshImport.onScanned(msg)
		return m, nil

	// --- Tunnel config loaded (for edit) ---
	case tunnelConfigLoadedMsg:
		if msg.err != nil || msg.config == nil {
			return m, nil
		}
		if m.editingTunnel != nil {
			cmd := m.form.showEdit(m.editingTunnel, msg.config)
			m.editingTunnel = nil
			return m, cmd
		}
		return m, nil

	// --- Tick ---
	case tickMsg:
		return m, tea.Batch(refreshStatus(m.clients), tickCmd())
	}

	return m, nil
}

func (m Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// Help overlay takes priority — only ? and Esc close it.
	if m.help.visible {
		if key.Matches(msg, m.keys.Help) || key.Matches(msg, m.keys.Escape) {
			m.help.toggle()
		}
		return m, nil
	}

	// Confirmation dialog.
	if m.confirm.visible {
		return m.handleConfirmKey(msg)
	}

	// Form dialog.
	if m.form.visible {
		return m.handleFormKey(msg)
	}

	// SSH import dialog.
	if m.sshImport.visible {
		return m.handleSSHImportKey(msg)
	}

	// Group input dialog.
	if m.groupIn.visible {
		return m.handleGroupInputKey(msg)
	}

	// Filter input active — delegate to filter.
	if m.filter.active {
		handled, cmd := m.filter.update(msg)
		if handled {
			m.tunnelList.textFilter = m.filter.matches
			m.tunnelList.applyFilter()
			return m, cmd
		}
	}

	// Esc clears filter if active.
	if key.Matches(msg, m.keys.Escape) {
		if m.filter.query != "" {
			m.filter.deactivate()
			m.tunnelList.textFilter = nil
			m.tunnelList.applyFilter()
			return m, nil
		}
		// Close log viewer if open.
		if m.logViewer.visible {
			m.logViewer.hide()
			m.tunnelList.height = m.height - 3
			return m, nil
		}
		return m, nil
	}

	// Global keys.
	if key.Matches(msg, m.keys.Quit) {
		return m, tea.Quit
	}

	if key.Matches(msg, m.keys.Help) {
		m.help.toggle()
		return m, nil
	}

	if key.Matches(msg, m.keys.Filter) {
		cmd := m.filter.activate()
		return m, cmd
	}

	if key.Matches(msg, m.keys.LogsAll) {
		if m.logViewer.visible && m.logViewer.tunnelName == "" {
			m.logViewer.hide()
			m.tunnelList.height = m.height - 3
		} else {
			logHeight := max((m.height-3)*30/100, 5)
			m.logViewer.width = m.width - m.sidebar.width - 2
			m.logViewer.height = logHeight
			m.logViewer.show("")
			m.tunnelList.height = m.height - 3 - logHeight
			return m, subscribeLogs(m.clients, "")
		}
		return m, nil
	}

	if key.Matches(msg, m.keys.Tab) {
		if m.focus == focusSidebar {
			m.focus = focusMain
			m.sidebar.focused = false
			m.tunnelList.focused = true
		} else {
			m.focus = focusSidebar
			m.sidebar.focused = true
			m.tunnelList.focused = false
		}
		return m, nil
	}

	// Import SSH config.
	if key.Matches(msg, m.keys.Import) {
		cmd := m.sshImport.show()
		return m, cmd
	}

	// Connect all / Disconnect all.
	if key.Matches(msg, m.keys.ConnectAll) {
		if m.focus == focusSidebar {
			if g := m.sidebar.selectedGroup(); g != "" {
				return m, connectGroup(m.clients, g)
			}
		}
		return m, connectAll(m.clients)
	}
	if key.Matches(msg, m.keys.DisconnectAll) {
		if m.focus == focusSidebar {
			if g := m.sidebar.selectedGroup(); g != "" {
				return m, disconnectGroup(m.clients, g)
			}
		}
		return m, disconnectAll(m.clients)
	}

	if m.focus == focusSidebar {
		return m.handleSidebarKey(msg)
	}
	return m.handleMainKey(msg)
}

func (m Model) handleSidebarKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, m.keys.Up):
		m.sidebar.moveUp()
		m.tunnelList.setTunnels(m.tunnelList.tunnels, m.sidebar.selectedGroup())
	case key.Matches(msg, m.keys.Down):
		m.sidebar.moveDown()
		m.tunnelList.setTunnels(m.tunnelList.tunnels, m.sidebar.selectedGroup())
	case key.Matches(msg, m.keys.Add):
		cmd := m.groupIn.show()
		return m, cmd
	case key.Matches(msg, m.keys.Rename):
		if g := m.sidebar.selectedGroup(); g != "" {
			cmd := m.groupIn.showRename(g)
			return m, cmd
		}
	case key.Matches(msg, m.keys.Delete):
		if g := m.sidebar.selectedGroup(); g != "" {
			m.confirm.show(confirmDeleteGroup, g)
		}
	}
	return m, nil
}

func (m Model) handleMainKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, m.keys.Up):
		m.tunnelList.moveUp()
	case key.Matches(msg, m.keys.Down):
		m.tunnelList.moveDown()
	case key.Matches(msg, m.keys.Connect):
		if t := m.tunnelList.selectedTunnel(); t != nil {
			return m, connectTunnel(m.clients, t.Name)
		}
	case key.Matches(msg, m.keys.Disconnect):
		if t := m.tunnelList.selectedTunnel(); t != nil {
			return m, disconnectTunnel(m.clients, t.Name)
		}
	case key.Matches(msg, m.keys.Retry):
		if t := m.tunnelList.selectedTunnel(); t != nil {
			return m, retryTunnel(m.clients, t.Name)
		}
	case key.Matches(msg, m.keys.Logs):
		if t := m.tunnelList.selectedTunnel(); t != nil {
			if m.logViewer.visible && m.logViewer.tunnelName == t.Name {
				m.logViewer.hide()
				m.tunnelList.height = m.height - 3
			} else {
				logHeight := max((m.height-3)*30/100, 5)
				m.logViewer.width = m.tunnelList.width
				m.logViewer.height = logHeight
				m.logViewer.show(t.Name)
				m.tunnelList.height = m.height - 3 - logHeight
				return m, subscribeLogs(m.clients, t.Name)
			}
		}
	case key.Matches(msg, m.keys.Delete):
		if t := m.tunnelList.selectedTunnel(); t != nil {
			m.confirm.show(confirmDeleteTunnel, t.Name)
		}
	case key.Matches(msg, m.keys.Add):
		cmd := m.form.showAdd()
		return m, cmd
	case key.Matches(msg, m.keys.Duplicate):
		if t := m.tunnelList.selectedTunnel(); t != nil {
			return m, duplicateTunnelConfig(t.Name)
		}
	case key.Matches(msg, m.keys.Edit):
		if t := m.tunnelList.selectedTunnel(); t != nil {
			m.editingTunnel = t
			return m, loadTunnelConfig(t.Name)
		}
	}
	return m, nil
}

func (m Model) handleConfirmKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "y", "Y":
		action := m.confirm.action
		target := m.confirm.target
		m.confirm.hide()
		switch action {
		case confirmDeleteTunnel:
			return m, deleteTunnelConfig(target)
		case confirmDeleteGroup:
			return m, deleteGroupConfig(target)
		}
	case "n", "N", "esc":
		m.confirm.hide()
	}
	return m, nil
}

func (m Model) handleFormKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// Check for submit before delegating.
	if msg.String() == "ctrl+s" {
		tc, group, errMsg := m.form.build()
		if errMsg != "" {
			m.form.err = errMsg
			return m, nil
		}
		if m.form.editing {
			return m, editTunnelConfig(m.form.original, group, tc)
		}
		return m, addTunnelConfig(group, tc)
	}

	handled, cmd := m.form.update(msg)
	if handled {
		return m, cmd
	}
	return m, nil
}

func (m Model) handleSSHImportKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// Check for submit before delegating.
	if msg.String() == "ctrl+s" || msg.String() == "enter" {
		if !m.sshImport.focusGrp && len(m.sshImport.selectedTunnels()) > 0 {
			tunnels := m.sshImport.selectedTunnels()
			group := m.sshImport.groupName()
			return m, importSSHTunnels(tunnels, group)
		}
	}

	handled, cmd := m.sshImport.update(msg)
	if handled {
		return m, cmd
	}
	return m, nil
}

func (m Model) handleGroupInputKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if msg.String() == "enter" {
		name := m.groupIn.value()
		if name != "" {
			if m.groupIn.renaming {
				oldName := m.groupIn.oldName
				m.groupIn.hide()
				return m, renameGroupConfig(oldName, name)
			}
			m.groupIn.hide()
			return m, addGroupConfig(name)
		}
		return m, nil
	}

	handled, cmd := m.groupIn.update(msg)
	if handled {
		return m, cmd
	}
	return m, nil
}

func (m Model) View() tea.View {
	if !m.ready {
		v := tea.NewView("Connecting to bore daemon...")
		v.AltScreen = true
		return v
	}

	// Title bar.
	activeTunnels := 0
	for _, t := range m.tunnelList.tunnels {
		if t.Status == borev1.TunnelStatus_TUNNEL_STATUS_ACTIVE {
			activeTunnels++
		}
	}
	connDot := m.styles.activeDot
	connLabel := "connected"
	if m.statusBar.status == nil {
		connDot = m.styles.errorDot
		connLabel = "disconnected"
	}
	titleRight := fmt.Sprintf("%d active  %s %s", activeTunnels, connDot, connLabel)
	titleGap := m.width - lipgloss.Width("Bore — SSH tunnel manager") - lipgloss.Width(titleRight) - 4
	if titleGap < 1 {
		titleGap = 1
	}
	title := m.styles.titleBar.Width(m.width).Render(
		fmt.Sprintf("%s%*s%s", "Bore — SSH tunnel manager", titleGap, "", titleRight),
	)

	// Sidebar.
	sidebarView := m.sidebar.view(m.styles)

	// Main panel: filter bar + tunnel list + optional log viewer.
	filterBar := m.filter.view(m.styles)
	mainContent := ""
	if filterBar != "" {
		mainContent = filterBar + "\n"
	}
	mainContent += m.tunnelList.view(m.styles)
	if m.logViewer.visible {
		mainContent += "\n" + m.logViewer.view(m.styles)
	}
	mainPanel := m.styles.mainPanel.
		Width(m.width - m.sidebar.width).
		Height(m.height - 3).
		Render(mainContent)

	// Content row.
	content := lipgloss.JoinHorizontal(lipgloss.Top, sidebarView, mainPanel)

	// Status bar.
	status := m.statusBar.view(m.styles)

	screen := lipgloss.JoinVertical(lipgloss.Left, title, content, status)

	// Overlay dialogs — render centered on top.
	if m.help.visible {
		screen = overlayCenter(screen, m.help.view(m.styles), m.width, m.height)
	}
	if m.confirm.visible {
		screen = overlayCenter(screen, m.confirm.view(m.styles), m.width, m.height)
	}
	if m.form.visible {
		screen = overlayCenter(screen, m.form.view(m.styles), m.width, m.height)
	}
	if m.sshImport.visible {
		screen = overlayCenter(screen, m.sshImport.view(m.styles), m.width, m.height)
	}
	if m.groupIn.visible {
		screen = overlayCenter(screen, m.groupIn.view(m.styles), m.width, m.height)
	}

	v := tea.NewView(screen)
	v.AltScreen = true
	return v
}

// overlayCenter places the overlay string centered over the background.
func overlayCenter(bg, overlay string, width, height int) string {
	bgLines := strings.Split(bg, "\n")
	overlayLines := strings.Split(overlay, "\n")
	overlayW := lipgloss.Width(overlay)

	x := (width - overlayW) / 2
	if x < 0 {
		x = 0
	}
	y := (height - len(overlayLines)) / 3
	if y < 0 {
		y = 0
	}

	// Pad background to ensure enough lines.
	for len(bgLines) < y+len(overlayLines) {
		bgLines = append(bgLines, strings.Repeat(" ", width))
	}

	for i, oLine := range overlayLines {
		row := y + i
		if row >= len(bgLines) {
			break
		}
		bgLine := bgLines[row]
		// Pad bgLine to at least x chars.
		bgRunes := []rune(bgLine)
		for len(bgRunes) < x {
			bgRunes = append(bgRunes, ' ')
		}
		oLineW := lipgloss.Width(oLine)
		endX := x + oLineW
		var trailing string
		if endX < len(bgRunes) {
			trailing = string(bgRunes[endX:])
		}
		bgLines[row] = string(bgRunes[:x]) + oLine + trailing
	}

	return strings.Join(bgLines, "\n")
}
