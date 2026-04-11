package tui

import "charm.land/bubbles/v2/key"

type keyMap struct {
	Up            key.Binding
	Down          key.Binding
	Connect       key.Binding
	Disconnect    key.Binding
	Retry         key.Binding
	Logs          key.Binding
	LogsAll       key.Binding
	Filter        key.Binding
	Tab           key.Binding
	Quit          key.Binding
	Help          key.Binding
	Delete        key.Binding
	Add           key.Binding
	Edit          key.Binding
	ConnectAll    key.Binding
	DisconnectAll key.Binding
	Duplicate     key.Binding
	Rename        key.Binding
	Import        key.Binding
	AutoRefresh   key.Binding
	Escape        key.Binding
}

func defaultKeyMap() keyMap {
	return keyMap{
		Up:            key.NewBinding(key.WithKeys("k", "up"), key.WithHelp("k/↑", "up")),
		Down:          key.NewBinding(key.WithKeys("j", "down"), key.WithHelp("j/↓", "down")),
		Connect:       key.NewBinding(key.WithKeys("c"), key.WithHelp("c", "connect")),
		Disconnect:    key.NewBinding(key.WithKeys("d"), key.WithHelp("d", "disconnect")),
		Retry:         key.NewBinding(key.WithKeys("r"), key.WithHelp("r", "retry")),
		Logs:          key.NewBinding(key.WithKeys("l"), key.WithHelp("l", "logs")),
		LogsAll:       key.NewBinding(key.WithKeys("L"), key.WithHelp("L", "all logs")),
		Filter:        key.NewBinding(key.WithKeys("/"), key.WithHelp("/", "filter")),
		Tab:           key.NewBinding(key.WithKeys("tab"), key.WithHelp("tab", "switch focus")),
		Quit:          key.NewBinding(key.WithKeys("q", "ctrl+c"), key.WithHelp("q", "quit")),
		Help:          key.NewBinding(key.WithKeys("?"), key.WithHelp("?", "help")),
		Delete:        key.NewBinding(key.WithKeys("x"), key.WithHelp("x", "delete")),
		Add:           key.NewBinding(key.WithKeys("a"), key.WithHelp("a", "add")),
		Edit:          key.NewBinding(key.WithKeys("e"), key.WithHelp("e", "edit")),
		ConnectAll:    key.NewBinding(key.WithKeys("C"), key.WithHelp("C", "connect all/group")),
		DisconnectAll: key.NewBinding(key.WithKeys("D"), key.WithHelp("D", "disconnect all/group")),
		Duplicate:     key.NewBinding(key.WithKeys("p"), key.WithHelp("p", "duplicate")),
		Rename:        key.NewBinding(key.WithKeys("r"), key.WithHelp("r", "rename")),
		Import:        key.NewBinding(key.WithKeys("i"), key.WithHelp("i", "import SSH")),
		AutoRefresh:   key.NewBinding(key.WithKeys("A"), key.WithHelp("A", "toggle auto-refresh")),
		Escape:        key.NewBinding(key.WithKeys("esc"), key.WithHelp("Esc", "close/clear")),
	}
}
