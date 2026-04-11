package tui

import (
	borev1 "github.com/hyperplex-tech/bore/gen/bore/v1"
	internalconfig "github.com/hyperplex-tech/bore/internal/config"
)

// --- Data fetch responses ---

type tunnelsLoadedMsg struct {
	tunnels []*borev1.Tunnel
	err     error
}

type groupsLoadedMsg struct {
	groups []*borev1.Group
	err    error
}

type daemonStatusMsg struct {
	status *borev1.StatusResponse
	err    error
}

// --- Event streaming ---

type eventStreamOpenedMsg struct {
	stream borev1.EventService_SubscribeClient
	err    error
}

type tunnelEventMsg struct {
	event *borev1.Event
	err   error
}

// --- Log streaming ---

type logStreamOpenedMsg struct {
	stream borev1.TunnelService_GetLogsClient
	err    error
}

type logEntryMsg struct {
	entry *borev1.LogEntry
	err   error
}

// --- Action results ---

type connectResultMsg struct {
	tunnels []*borev1.Tunnel
	err     error
}

type disconnectResultMsg struct {
	tunnels []*borev1.Tunnel
	err     error
}

type retryResultMsg struct {
	tunnels []*borev1.Tunnel
	err     error
}

// --- Config changes ---

type configChangedMsg struct {
	err     error
	message string
}

type daemonReloadedMsg struct{}

type tunnelConfigLoadedMsg struct {
	config *internalconfig.TunnelConfig
	err    error
}

// --- UI ---

type tickMsg struct{}
