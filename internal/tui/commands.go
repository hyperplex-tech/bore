package tui

import (
	"context"
	"fmt"
	"time"

	tea "charm.land/bubbletea/v2"

	borev1 "github.com/hyperplex-tech/bore/gen/bore/v1"
	"github.com/hyperplex-tech/bore/internal/cli"
	internalconfig "github.com/hyperplex-tech/bore/internal/config"
)

func refreshTunnels(clients *cli.Clients) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		resp, err := clients.Tunnel.List(ctx, &borev1.ListRequest{})
		if err != nil {
			return tunnelsLoadedMsg{err: err}
		}
		return tunnelsLoadedMsg{tunnels: resp.Tunnels}
	}
}

func refreshGroups(clients *cli.Clients) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		resp, err := clients.Group.ListGroups(ctx, &borev1.ListGroupsRequest{})
		if err != nil {
			return groupsLoadedMsg{err: err}
		}
		return groupsLoadedMsg{groups: resp.Groups}
	}
}

func refreshStatus(clients *cli.Clients) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		resp, err := clients.Daemon.Status(ctx, &borev1.StatusRequest{})
		if err != nil {
			return daemonStatusMsg{err: err}
		}
		return daemonStatusMsg{status: resp}
	}
}

func subscribeEvents(clients *cli.Clients) tea.Cmd {
	return func() tea.Msg {
		stream, err := clients.Event.Subscribe(context.Background(), &borev1.SubscribeRequest{})
		if err != nil {
			return eventStreamOpenedMsg{err: err}
		}
		return eventStreamOpenedMsg{stream: stream}
	}
}

func waitForEvent(stream borev1.EventService_SubscribeClient) tea.Cmd {
	return func() tea.Msg {
		event, err := stream.Recv()
		if err != nil {
			return tunnelEventMsg{err: err}
		}
		return tunnelEventMsg{event: event}
	}
}

func connectTunnel(clients *cli.Clients, name string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		resp, err := clients.Tunnel.Connect(ctx, &borev1.ConnectRequest{Names: []string{name}})
		if err != nil {
			return connectResultMsg{err: err}
		}
		return connectResultMsg{tunnels: resp.Tunnels}
	}
}

func disconnectTunnel(clients *cli.Clients, name string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		resp, err := clients.Tunnel.Disconnect(ctx, &borev1.DisconnectRequest{Names: []string{name}})
		if err != nil {
			return disconnectResultMsg{err: err}
		}
		return disconnectResultMsg{tunnels: resp.Tunnels}
	}
}

func retryTunnel(clients *cli.Clients, name string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		resp, err := clients.Tunnel.Connect(ctx, &borev1.ConnectRequest{Names: []string{name}})
		if err != nil {
			return retryResultMsg{err: err}
		}
		return retryResultMsg{tunnels: resp.Tunnels}
	}
}

func connectAll(clients *cli.Clients) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		resp, err := clients.Tunnel.Connect(ctx, &borev1.ConnectRequest{})
		if err != nil {
			return connectResultMsg{err: err}
		}
		return connectResultMsg{tunnels: resp.Tunnels}
	}
}

func disconnectAll(clients *cli.Clients) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		resp, err := clients.Tunnel.Disconnect(ctx, &borev1.DisconnectRequest{})
		if err != nil {
			return disconnectResultMsg{err: err}
		}
		return disconnectResultMsg{tunnels: resp.Tunnels}
	}
}

func connectGroup(clients *cli.Clients, group string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		resp, err := clients.Tunnel.Connect(ctx, &borev1.ConnectRequest{Group: group})
		if err != nil {
			return connectResultMsg{err: err}
		}
		return connectResultMsg{tunnels: resp.Tunnels}
	}
}

func disconnectGroup(clients *cli.Clients, group string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		resp, err := clients.Tunnel.Disconnect(ctx, &borev1.DisconnectRequest{Group: group})
		if err != nil {
			return disconnectResultMsg{err: err}
		}
		return disconnectResultMsg{tunnels: resp.Tunnels}
	}
}

func deleteTunnelConfig(name string) tea.Cmd {
	return func() tea.Msg {
		configPath := internalconfig.ConfigPath()
		err := internalconfig.RemoveTunnel(configPath, name)
		return configChangedMsg{err: err}
	}
}

func duplicateTunnelConfig(name string) tea.Cmd {
	return func() tea.Msg {
		configPath := internalconfig.ConfigPath()
		err := internalconfig.DuplicateTunnel(configPath, name)
		return configChangedMsg{err: err, message: fmt.Sprintf("Duplicated tunnel %q", name)}
	}
}

func addTunnelConfig(group string, tc internalconfig.TunnelConfig) tea.Cmd {
	return func() tea.Msg {
		configPath := internalconfig.ConfigPath()
		err := internalconfig.AddTunnel(configPath, group, tc)
		return configChangedMsg{err: err, message: fmt.Sprintf("Added tunnel %q", tc.Name)}
	}
}

func editTunnelConfig(originalName string, group string, tc internalconfig.TunnelConfig) tea.Cmd {
	return func() tea.Msg {
		configPath := internalconfig.ConfigPath()
		err := internalconfig.UpdateTunnel(configPath, originalName, tc, group)
		return configChangedMsg{err: err, message: fmt.Sprintf("Updated tunnel %q", tc.Name)}
	}
}

func addGroupConfig(name string) tea.Cmd {
	return func() tea.Msg {
		configPath := internalconfig.ConfigPath()
		err := internalconfig.AddGroup(configPath, name, "")
		return configChangedMsg{err: err, message: fmt.Sprintf("Created group %q", name)}
	}
}

func deleteGroupConfig(name string) tea.Cmd {
	return func() tea.Msg {
		configPath := internalconfig.ConfigPath()
		err := internalconfig.RemoveGroup(configPath, name)
		return configChangedMsg{err: err}
	}
}

func renameGroupConfig(oldName, newName string) tea.Cmd {
	return func() tea.Msg {
		configPath := internalconfig.ConfigPath()
		err := internalconfig.RenameGroup(configPath, oldName, newName)
		return configChangedMsg{err: err, message: fmt.Sprintf("Renamed group %q to %q", oldName, newName)}
	}
}

func importSSHTunnels(tunnels []internalconfig.TunnelConfig, group string) tea.Cmd {
	return func() tea.Msg {
		configPath := internalconfig.ConfigPath()
		imported := 0
		for _, tc := range tunnels {
			if err := internalconfig.AddTunnel(configPath, group, tc); err != nil {
				continue
			}
			imported++
		}
		return configChangedMsg{message: fmt.Sprintf("Imported %d tunnel(s) into group %q", imported, group)}
	}
}

func reloadDaemonConfig(clients *cli.Clients) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		clients.Daemon.ReloadConfig(ctx, &borev1.ReloadConfigRequest{})
		return daemonReloadedMsg{}
	}
}

func subscribeLogs(clients *cli.Clients, tunnelName string) tea.Cmd {
	return func() tea.Msg {
		stream, err := clients.Tunnel.GetLogs(context.Background(), &borev1.GetLogsRequest{
			Name:   tunnelName,
			Tail:   50,
			Follow: true,
		})
		if err != nil {
			return logStreamOpenedMsg{err: err}
		}
		return logStreamOpenedMsg{stream: stream}
	}
}

func waitForLogEntry(stream borev1.TunnelService_GetLogsClient) tea.Cmd {
	return func() tea.Msg {
		entry, err := stream.Recv()
		if err != nil {
			return logEntryMsg{err: err}
		}
		return logEntryMsg{entry: entry}
	}
}

func tickCmd() tea.Cmd {
	return func() tea.Msg {
		time.Sleep(10 * time.Second)
		return tickMsg{}
	}
}

func delayedResubscribe(clients *cli.Clients) tea.Cmd {
	return func() tea.Msg {
		time.Sleep(2 * time.Second)
		stream, err := clients.Event.Subscribe(context.Background(), &borev1.SubscribeRequest{})
		if err != nil {
			return eventStreamOpenedMsg{err: err}
		}
		return eventStreamOpenedMsg{stream: stream}
	}
}

func loadTunnelConfig(name string) tea.Cmd {
	return func() tea.Msg {
		configPath := internalconfig.ConfigPath()
		cfg, err := internalconfig.LoadOrDefault(configPath)
		if err != nil {
			return tunnelConfigLoadedMsg{err: err}
		}
		rt, found := cfg.FindTunnel(name)
		if !found {
			return tunnelConfigLoadedMsg{err: fmt.Errorf("tunnel %q not found", name)}
		}
		tc := rt.TunnelConfig
		return tunnelConfigLoadedMsg{config: &tc}
	}
}
