//go:build desktop

package desktop

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	goruntime "runtime"
	"time"

	"github.com/rs/zerolog/log"
	wailsruntime "github.com/wailsapp/wails/v2/pkg/runtime"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	borev1 "github.com/hyperplex-tech/bore/gen/bore/v1"
	"github.com/hyperplex-tech/bore/internal/config"
	"github.com/hyperplex-tech/bore/internal/ipc"
	"github.com/hyperplex-tech/bore/internal/profile"
)

// App is the Wails application backend. It wraps gRPC calls to the bore daemon.
type App struct {
	ctx    context.Context
	conn   *grpc.ClientConn
	daemon borev1.DaemonServiceClient
	tunnel borev1.TunnelServiceClient
	group  borev1.GroupServiceClient
	event  borev1.EventServiceClient
}

// NewApp creates a new App.
func NewApp() *App {
	return &App{}
}

// Startup is called by Wails at application startup.
func (a *App) Startup(ctx context.Context) {
	a.ctx = ctx

	// Ensure config/data directories exist.
	if err := config.EnsureDirs(); err != nil {
		log.Error().Err(err).Msg("failed to create directories")
	}

	if err := a.connect(); err != nil {
		log.Warn().Err(err).Msg("daemon not reachable, attempting to start it")
		if startErr := a.startDaemon(); startErr != nil {
			log.Error().Err(startErr).Msg("failed to auto-start daemon")
		} else {
			// Retry connection with backoff — daemon may need time to open its socket.
			var connectErr error
			for _, delay := range []time.Duration{500 * time.Millisecond, 1 * time.Second, 2 * time.Second, 3 * time.Second} {
				time.Sleep(delay)
				if connectErr = a.connect(); connectErr == nil {
					break
				}
				log.Debug().Err(connectErr).Dur("retry_in", delay).Msg("daemon not ready yet, retrying")
			}
			if connectErr != nil {
				log.Error().Err(connectErr).Msg("failed to connect after starting daemon")
			}
		}
	}
}

// Shutdown is called by Wails at application shutdown.
func (a *App) Shutdown(ctx context.Context) {
	if a.conn != nil {
		a.conn.Close()
	}
}

func (a *App) connect() error {
	socketPath := os.Getenv("BORE_SOCKET")
	if socketPath == "" {
		socketPath = config.SocketPath()
	}
	conn, err := grpc.NewClient(
		ipc.DialTarget(socketPath),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		ipc.DialOption(),
	)
	if err != nil {
		return fmt.Errorf("connecting to daemon at %s: %w", socketPath, err)
	}
	a.conn = conn
	a.daemon = borev1.NewDaemonServiceClient(conn)
	a.tunnel = borev1.NewTunnelServiceClient(conn)
	a.group = borev1.NewGroupServiceClient(conn)
	a.event = borev1.NewEventServiceClient(conn)

	// Verify the daemon is actually reachable.
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if _, err := a.daemon.Status(ctx, &borev1.StatusRequest{}); err != nil {
		a.conn.Close()
		a.conn = nil
		a.daemon = nil
		a.tunnel = nil
		a.group = nil
		a.event = nil
		return fmt.Errorf("daemon not responding: %w", err)
	}

	// Start event subscription in background.
	go a.subscribeEvents()

	return nil
}

// startDaemon tries to launch the bored daemon process.
func (a *App) startDaemon() error {
	// Try platform service manager first.
	if started := a.startDaemonViaService(); started {
		return nil
	}

	// Fall back to starting bored directly.
	boredName := "bored"
	if goruntime.GOOS == "windows" {
		boredName = "bored.exe"
	}

	boredPath, err := exec.LookPath(boredName)
	if err != nil {
		// Try next to our own binary, then in the data directory.
		candidates := []string{}
		if self, selfErr := os.Executable(); selfErr == nil {
			candidates = append(candidates, filepath.Join(filepath.Dir(self), boredName))
		}
		candidates = append(candidates, filepath.Join(config.DataDir(), boredName))

		found := false
		for _, candidate := range candidates {
			if _, statErr := os.Stat(candidate); statErr == nil {
				boredPath = candidate
				found = true
				break
			}
		}
		if !found {
			return fmt.Errorf("bored binary not found — install with: make install")
		}
	}

	cmd := exec.Command(boredPath)
	cmd.Env = os.Environ()
	hideDaemonConsole(cmd)
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("starting daemon: %w", err)
	}

	log.Info().Int("pid", cmd.Process.Pid).Msg("started daemon process")
	return nil
}

// startDaemonViaService tries the platform's service manager. Returns true if successful.
func (a *App) startDaemonViaService() bool {
	switch goruntime.GOOS {
	case "linux":
		if path, err := exec.LookPath("systemctl"); err == nil {
			cmd := exec.Command(path, "--user", "start", "bored")
			if err := cmd.Run(); err == nil {
				log.Info().Msg("started daemon via systemd")
				return true
			}
		}
	case "darwin":
		if path, err := exec.LookPath("launchctl"); err == nil {
			cmd := exec.Command(path, "start", "com.bore.daemon")
			if err := cmd.Run(); err == nil {
				log.Info().Msg("started daemon via launchd")
				return true
			}
		}
	case "windows":
		cmd := exec.Command("schtasks", "/run", "/tn", "Bore Daemon")
		if err := cmd.Run(); err == nil {
			log.Info().Msg("started daemon via task scheduler")
			time.Sleep(1 * time.Second)
			return true
		}
	}
	return false
}

// --- Exposed methods (called from React via Wails bindings) ---

// TunnelInfo is the JSON-friendly tunnel representation for the frontend.
type TunnelInfo struct {
	Name          string `json:"name"`
	Group         string `json:"group"`
	Type          string `json:"type"`
	Status        string `json:"status"`
	LocalPort     int    `json:"localPort"`
	LocalHost     string `json:"localHost"`
	RemoteHost    string `json:"remoteHost"`
	RemotePort    int    `json:"remotePort"`
	SSHHost       string `json:"sshHost"`
	SSHPort       int    `json:"sshPort"`
	SSHUser       string `json:"sshUser"`
	AuthMethod    string `json:"authMethod"`
	IdentityFile  string `json:"identityFile"`
	JumpHosts     []string `json:"jumpHosts"`
	K8sContext    string   `json:"k8sContext"`
	K8sNamespace  string   `json:"k8sNamespace"`
	K8sResource   string   `json:"k8sResource"`
	PreConnect    string   `json:"preConnect"`
	PostConnect   string   `json:"postConnect"`
	Reconnect     *bool    `json:"reconnect"`
	ErrorMessage  string   `json:"errorMessage"`
	ConnectedAt   string `json:"connectedAt"`
	LastErrorAt   string `json:"lastErrorAt"`
	RetryCount    int    `json:"retryCount"`
	NextRetrySecs int    `json:"nextRetrySecs"`
}

// GroupInfo is the JSON-friendly group representation for the frontend.
type GroupInfo struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	TunnelCount int    `json:"tunnelCount"`
	ActiveCount int    `json:"activeCount"`
}

// StatusInfo is the JSON-friendly daemon status for the frontend.
type StatusInfo struct {
	Version            string `json:"version"`
	ActiveTunnels      int    `json:"activeTunnels"`
	TotalTunnels       int    `json:"totalTunnels"`
	SocketPath         string `json:"socketPath"`
	ConfigPath         string `json:"configPath"`
	SSHAgentAvailable  bool   `json:"sshAgentAvailable"`
	SSHAgentKeys       int    `json:"sshAgentKeys"`
	Connected          bool   `json:"connected"`
	TailscaleAvailable bool   `json:"tailscaleAvailable"`
	TailscaleConnected bool   `json:"tailscaleConnected"`
	TailscaleIP        string `json:"tailscaleIp"`
	TailscaleHostname  string `json:"tailscaleHostname"`
}

// GetStatus returns the daemon status.
func (a *App) GetStatus() StatusInfo {
	if a.daemon == nil {
		return StatusInfo{Connected: false}
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	resp, err := a.daemon.Status(ctx, &borev1.StatusRequest{})
	if err != nil {
		return StatusInfo{Connected: false}
	}
	return StatusInfo{
		Version:            resp.Version,
		ActiveTunnels:      int(resp.ActiveTunnels),
		TotalTunnels:       int(resp.TotalTunnels),
		SocketPath:         resp.SocketPath,
		ConfigPath:         resp.ConfigPath,
		SSHAgentAvailable:  resp.SshAgentAvailable,
		SSHAgentKeys:       int(resp.SshAgentKeys),
		Connected:          true,
		TailscaleAvailable: resp.TailscaleAvailable,
		TailscaleConnected: resp.TailscaleConnected,
		TailscaleIP:        resp.TailscaleIp,
		TailscaleHostname:  resp.TailscaleHostname,
	}
}

// ListTunnels returns all tunnels, optionally filtered by group.
func (a *App) ListTunnels(group string) []TunnelInfo {
	if a.tunnel == nil {
		return nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	resp, err := a.tunnel.List(ctx, &borev1.ListRequest{Group: group})
	if err != nil {
		log.Error().Err(err).Msg("list tunnels failed")
		return nil
	}
	return protoTunnelsToInfo(resp.Tunnels)
}

// ListGroups returns all groups.
func (a *App) ListGroups() []GroupInfo {
	if a.group == nil {
		return nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	resp, err := a.group.ListGroups(ctx, &borev1.ListGroupsRequest{})
	if err != nil {
		log.Error().Err(err).Msg("list groups failed")
		return nil
	}
	var groups []GroupInfo
	for _, g := range resp.Groups {
		groups = append(groups, GroupInfo{
			Name:        g.Name,
			Description: g.Description,
			TunnelCount: int(g.TunnelCount),
			ActiveCount: int(g.ActiveCount),
		})
	}
	return groups
}

// ConnectTunnels connects tunnels by name or group.
func (a *App) ConnectTunnels(names []string, group string) []TunnelInfo {
	if a.tunnel == nil {
		return nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	resp, err := a.tunnel.Connect(ctx, &borev1.ConnectRequest{Names: names, Group: group})
	if err != nil {
		log.Error().Err(err).Msg("connect failed")
		return nil
	}
	return protoTunnelsToInfo(resp.Tunnels)
}

// DisconnectTunnels disconnects tunnels by name or group.
func (a *App) DisconnectTunnels(names []string, group string) []TunnelInfo {
	if a.tunnel == nil {
		return nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	resp, err := a.tunnel.Disconnect(ctx, &borev1.DisconnectRequest{Names: names, Group: group})
	if err != nil {
		log.Error().Err(err).Msg("disconnect failed")
		return nil
	}
	return protoTunnelsToInfo(resp.Tunnels)
}

// ConnectAll connects all tunnels.
func (a *App) ConnectAll() []TunnelInfo {
	return a.ConnectTunnels(nil, "")
}

// DisconnectAll disconnects all tunnels.
func (a *App) DisconnectAll() []TunnelInfo {
	return a.DisconnectTunnels(nil, "")
}

// RetryTunnel retries a single errored tunnel.
func (a *App) RetryTunnel(name string) []TunnelInfo {
	return a.ConnectTunnels([]string{name}, "")
}

// PauseTunnel disconnects a single tunnel.
func (a *App) PauseTunnel(name string) []TunnelInfo {
	return a.DisconnectTunnels([]string{name}, "")
}

// AddTunnelRequest is the JSON-friendly request for adding a tunnel.
type AddTunnelRequest struct {
	Name         string   `json:"name"`
	Group        string   `json:"group"`
	Type         string   `json:"type"`
	LocalHost    string   `json:"localHost"`
	LocalPort    int      `json:"localPort"`
	RemoteHost   string   `json:"remoteHost"`
	RemotePort   int      `json:"remotePort"`
	SSHHost      string   `json:"sshHost"`
	SSHPort      int      `json:"sshPort"`
	SSHUser      string   `json:"sshUser"`
	AuthMethod   string   `json:"authMethod"`
	IdentityFile string   `json:"identityFile"`
	JumpHosts    []string `json:"jumpHosts"`
	K8sContext   string   `json:"k8sContext"`
	K8sNamespace string   `json:"k8sNamespace"`
	K8sResource  string   `json:"k8sResource"`
	PreConnect   string   `json:"preConnect"`
	PostConnect  string   `json:"postConnect"`
	Reconnect    *bool    `json:"reconnect"`
}

// AddTunnel adds a new tunnel to the config and reloads the daemon.
func (a *App) AddTunnel(req AddTunnelRequest) string {
	if req.Name == "" || req.LocalPort == 0 {
		return "Missing required fields (name, localPort)"
	}

	tunnelType := req.Type
	if tunnelType == "" {
		tunnelType = "local"
	}

	if tunnelType == "k8s" {
		if req.K8sResource == "" || req.RemotePort == 0 {
			return "K8s tunnels require resource and remotePort"
		}
	} else if tunnelType == "dynamic" {
		if req.SSHHost == "" {
			return "SOCKS5 tunnels require sshHost"
		}
	} else {
		if req.RemoteHost == "" || req.RemotePort == 0 || req.SSHHost == "" {
			return "SSH tunnels require remoteHost, remotePort, and sshHost"
		}
	}

	group := req.Group
	if group == "" {
		group = "default"
	}

	tc := config.TunnelConfig{
		Name:         req.Name,
		Type:         tunnelType,
		LocalHost:    req.LocalHost,
		LocalPort:    req.LocalPort,
		RemoteHost:   req.RemoteHost,
		RemotePort:   req.RemotePort,
		SSHHost:      req.SSHHost,
		SSHPort:      req.SSHPort,
		SSHUser:      req.SSHUser,
		AuthMethod:   req.AuthMethod,
		IdentityFile: req.IdentityFile,
		JumpHosts:    req.JumpHosts,
		K8sContext:   req.K8sContext,
		K8sNamespace: req.K8sNamespace,
		K8sResource:  req.K8sResource,
		Reconnect:    req.Reconnect,
	}
	if req.PreConnect != "" || req.PostConnect != "" {
		tc.Hooks = &config.Hooks{
			PreConnect:  req.PreConnect,
			PostConnect: req.PostConnect,
		}
	}

	configPath := config.ConfigPath()
	if err := config.AddTunnel(configPath, group, tc); err != nil {
		return fmt.Sprintf("Failed to add tunnel: %v", err)
	}

	// Reload daemon config.
	if a.daemon != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		a.daemon.ReloadConfig(ctx, &borev1.ReloadConfigRequest{})
	}

	return "" // empty string = success
}

// EditTunnelRequest is the JSON-friendly request for editing a tunnel.
type EditTunnelRequest struct {
	OriginalName string   `json:"originalName"`
	Name         string   `json:"name"`
	Group        string   `json:"group"`
	Type         string   `json:"type"`
	LocalHost    string   `json:"localHost"`
	LocalPort    int      `json:"localPort"`
	RemoteHost   string   `json:"remoteHost"`
	RemotePort   int      `json:"remotePort"`
	SSHHost      string   `json:"sshHost"`
	SSHPort      int      `json:"sshPort"`
	SSHUser      string   `json:"sshUser"`
	AuthMethod   string   `json:"authMethod"`
	IdentityFile string   `json:"identityFile"`
	JumpHosts    []string `json:"jumpHosts"`
	K8sContext   string   `json:"k8sContext"`
	K8sNamespace string   `json:"k8sNamespace"`
	K8sResource  string   `json:"k8sResource"`
	PreConnect   string   `json:"preConnect"`
	PostConnect  string   `json:"postConnect"`
	Reconnect    *bool    `json:"reconnect"`
}

// EditTunnel updates an existing tunnel in the config and reloads the daemon.
func (a *App) EditTunnel(req EditTunnelRequest) string {
	if req.Name == "" || req.LocalPort == 0 {
		return "Missing required fields (name, localPort)"
	}

	tunnelType := req.Type
	if tunnelType == "" {
		tunnelType = "local"
	}

	if tunnelType == "k8s" {
		if req.K8sResource == "" || req.RemotePort == 0 {
			return "K8s tunnels require resource and remotePort"
		}
	} else if tunnelType == "dynamic" {
		if req.SSHHost == "" {
			return "SOCKS5 tunnels require sshHost"
		}
	} else {
		if req.RemoteHost == "" || req.RemotePort == 0 || req.SSHHost == "" {
			return "SSH tunnels require remoteHost, remotePort, and sshHost"
		}
	}

	group := req.Group
	if group == "" {
		group = "default"
	}

	tc := config.TunnelConfig{
		Name:         req.Name,
		Type:         tunnelType,
		LocalHost:    req.LocalHost,
		LocalPort:    req.LocalPort,
		RemoteHost:   req.RemoteHost,
		RemotePort:   req.RemotePort,
		SSHHost:      req.SSHHost,
		SSHPort:      req.SSHPort,
		SSHUser:      req.SSHUser,
		AuthMethod:   req.AuthMethod,
		IdentityFile: req.IdentityFile,
		JumpHosts:    req.JumpHosts,
		K8sContext:   req.K8sContext,
		K8sNamespace: req.K8sNamespace,
		K8sResource:  req.K8sResource,
		Reconnect:    req.Reconnect,
	}
	if req.PreConnect != "" || req.PostConnect != "" {
		tc.Hooks = &config.Hooks{
			PreConnect:  req.PreConnect,
			PostConnect: req.PostConnect,
		}
	}

	configPath := config.ConfigPath()
	if err := config.UpdateTunnel(configPath, req.OriginalName, tc, group); err != nil {
		return fmt.Sprintf("Failed to edit tunnel: %v", err)
	}

	if a.daemon != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		a.daemon.ReloadConfig(ctx, &borev1.ReloadConfigRequest{})
	}

	return ""
}

// DuplicateTunnel copies a tunnel with " - Copy" appended to the name.
func (a *App) DuplicateTunnel(name string) string {
	configPath := config.ConfigPath()
	if err := config.DuplicateTunnel(configPath, name); err != nil {
		return fmt.Sprintf("Failed to duplicate tunnel: %v", err)
	}
	if a.daemon != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		a.daemon.ReloadConfig(ctx, &borev1.ReloadConfigRequest{})
	}
	return ""
}

// DeleteTunnel removes a tunnel from the config and reloads the daemon.
func (a *App) DeleteTunnel(name string) string {
	configPath := config.ConfigPath()
	if err := config.RemoveTunnel(configPath, name); err != nil {
		return fmt.Sprintf("Failed to delete tunnel: %v", err)
	}
	if a.daemon != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		a.daemon.ReloadConfig(ctx, &borev1.ReloadConfigRequest{})
	}
	return ""
}

// TunnelConfigDetails returns the full config for a tunnel (including hooks).
// Used by the edit dialog to pre-fill fields not in the proto.
func (a *App) GetTunnelConfig(name string) map[string]interface{} {
	configPath := config.ConfigPath()
	cfg, err := config.LoadOrDefault(configPath)
	if err != nil {
		return nil
	}
	rt, found := cfg.FindTunnel(name)
	if !found {
		return nil
	}
	result := map[string]interface{}{
		"preConnect":  "",
		"postConnect": "",
		"reconnect":   rt.Reconnect,
	}
	if rt.Hooks != nil {
		result["preConnect"] = rt.Hooks.PreConnect
		result["postConnect"] = rt.Hooks.PostConnect
	}
	return result
}

// LogEntry is a JSON-friendly log entry for the frontend.
type LogEntry struct {
	Timestamp  string `json:"timestamp"`
	Level      string `json:"level"`
	Message    string `json:"message"`
	TunnelName string `json:"tunnelName"`
}

// GetTunnelLogs fetches recent log entries for a tunnel.
func (a *App) GetTunnelLogs(name string, tail int) []LogEntry {
	if a.tunnel == nil {
		return nil
	}
	if tail <= 0 {
		tail = 200
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	stream, err := a.tunnel.GetLogs(ctx, &borev1.GetLogsRequest{
		Name:   name,
		Tail:   int32(tail),
		Follow: false,
	})
	if err != nil {
		log.Error().Err(err).Str("tunnel", name).Msg("get logs failed")
		return nil
	}

	var entries []LogEntry
	for {
		entry, err := stream.Recv()
		if err != nil {
			break
		}
		ts := ""
		if entry.Timestamp != nil {
			ts = entry.Timestamp.AsTime().Format("15:04:05")
		}
		entries = append(entries, LogEntry{
			Timestamp:  ts,
			Level:      entry.Level,
			Message:    entry.Message,
			TunnelName: entry.TunnelName,
		})
	}
	return entries
}

// GetAllLogs fetches recent log entries across all tunnels.
func (a *App) GetAllLogs(tail int) []LogEntry {
	return a.GetTunnelLogs("", tail)
}

// AddGroup creates a new empty group.
func (a *App) AddGroup(name string, description string) string {
	configPath := config.ConfigPath()
	if err := config.AddGroup(configPath, name, description); err != nil {
		return fmt.Sprintf("Failed to add group: %v", err)
	}
	if a.daemon != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		a.daemon.ReloadConfig(ctx, &borev1.ReloadConfigRequest{})
	}
	return ""
}

// RenameGroup renames a group.
func (a *App) RenameGroup(oldName string, newName string) string {
	configPath := config.ConfigPath()
	if err := config.RenameGroup(configPath, oldName, newName); err != nil {
		return fmt.Sprintf("Failed to rename group: %v", err)
	}
	if a.daemon != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		a.daemon.ReloadConfig(ctx, &borev1.ReloadConfigRequest{})
	}
	return ""
}

// DeleteGroup removes an empty group.
func (a *App) DeleteGroup(name string) string {
	configPath := config.ConfigPath()
	if err := config.RemoveGroup(configPath, name); err != nil {
		return fmt.Sprintf("Failed to delete group: %v", err)
	}
	if a.daemon != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		a.daemon.ReloadConfig(ctx, &borev1.ReloadConfigRequest{})
	}
	return ""
}

// SSHImportEntry is a discovered tunnel from SSH config, shown to the user for confirmation.
type SSHImportEntry struct {
	Name       string `json:"name"`
	LocalPort  int    `json:"localPort"`
	RemoteHost string `json:"remoteHost"`
	RemotePort int    `json:"remotePort"`
	SSHHost    string `json:"sshHost"`
	SSHUser    string `json:"sshUser"`
}

// PreviewSSHImport parses ~/.ssh/config and returns discovered tunnels.
func (a *App) PreviewSSHImport() []SSHImportEntry {
	hosts, err := profile.ImportSSHConfig("")
	if err != nil {
		log.Error().Err(err).Msg("SSH config import failed")
		return nil
	}

	tunnels := profile.ToTunnelConfigs(hosts)
	var entries []SSHImportEntry
	for _, tc := range tunnels {
		entries = append(entries, SSHImportEntry{
			Name:       tc.Name,
			LocalPort:  tc.LocalPort,
			RemoteHost: tc.RemoteHost,
			RemotePort: tc.RemotePort,
			SSHHost:    tc.SSHHost,
			SSHUser:    tc.SSHUser,
		})
	}
	return entries
}

// ImportSSHTunnels imports selected tunnels from SSH config into bore config.
func (a *App) ImportSSHTunnels(names []string, group string) string {
	if group == "" {
		group = "imported"
	}

	hosts, err := profile.ImportSSHConfig("")
	if err != nil {
		return fmt.Sprintf("Failed to parse SSH config: %v", err)
	}

	tunnels := profile.ToTunnelConfigs(hosts)
	nameSet := make(map[string]bool)
	for _, n := range names {
		nameSet[n] = true
	}

	configPath := config.ConfigPath()
	imported := 0
	for _, tc := range tunnels {
		if len(names) > 0 && !nameSet[tc.Name] {
			continue
		}
		if err := config.AddTunnel(configPath, group, tc); err != nil {
			log.Warn().Err(err).Str("tunnel", tc.Name).Msg("skip import")
			continue
		}
		imported++
	}

	// Reload daemon.
	if a.daemon != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		a.daemon.ReloadConfig(ctx, &borev1.ReloadConfigRequest{})
	}

	if imported == 0 {
		return "No new tunnels imported (may already exist)"
	}
	return "" // success
}

// subscribeEvents subscribes to daemon events and emits them to the frontend.
func (a *App) subscribeEvents() {
	for {
		if a.event == nil {
			time.Sleep(2 * time.Second)
			continue
		}

		stream, err := a.event.Subscribe(context.Background(), &borev1.SubscribeRequest{})
		if err != nil {
			log.Debug().Err(err).Msg("event subscribe failed, retrying...")
			time.Sleep(2 * time.Second)
			continue
		}

		for {
			evt, err := stream.Recv()
			if err != nil {
				log.Debug().Err(err).Msg("event stream broken, reconnecting...")
				break
			}

			// Emit to the Wails frontend.
			wailsruntime.EventsEmit(a.ctx, "tunnel-event", map[string]interface{}{
				"type":       evt.Type.String(),
				"tunnelName": evt.TunnelName,
				"message":    evt.Message,
				"timestamp":  evt.Timestamp.AsTime().Format(time.RFC3339),
			})
		}

		time.Sleep(1 * time.Second)
	}
}

// --- Proto conversion helpers ---

func protoTunnelsToInfo(tunnels []*borev1.Tunnel) []TunnelInfo {
	var result []TunnelInfo
	for _, t := range tunnels {
		info := TunnelInfo{
			Name:          t.Name,
			Group:         t.Group,
			Type:          protoTunnelType(t.Type),
			Status:        protoTunnelStatus(t.Status),
			LocalPort:     int(t.LocalPort),
			LocalHost:     t.LocalHost,
			RemoteHost:    t.RemoteHost,
			RemotePort:    int(t.RemotePort),
			SSHHost:       t.SshHost,
			SSHPort:       int(t.SshPort),
			SSHUser:       t.SshUser,
			AuthMethod:    t.AuthMethod,
			IdentityFile:  t.IdentityFile,
			JumpHosts:     t.JumpHosts,
			K8sContext:    t.K8SContext,
			K8sNamespace:  t.K8SNamespace,
			K8sResource:   t.K8SResource,
			ErrorMessage:  t.ErrorMessage,
			RetryCount:    int(t.RetryCount),
			NextRetrySecs: int(t.NextRetrySecs),
		}
		if t.ConnectedAt != nil {
			info.ConnectedAt = t.ConnectedAt.AsTime().Format(time.RFC3339)
		}
		if t.LastErrorAt != nil {
			info.LastErrorAt = t.LastErrorAt.AsTime().Format(time.RFC3339)
		}
		result = append(result, info)
	}
	return result
}

func protoTunnelStatus(s borev1.TunnelStatus) string {
	switch s {
	case borev1.TunnelStatus_TUNNEL_STATUS_ACTIVE:
		return "active"
	case borev1.TunnelStatus_TUNNEL_STATUS_CONNECTING:
		return "connecting"
	case borev1.TunnelStatus_TUNNEL_STATUS_ERROR:
		return "error"
	case borev1.TunnelStatus_TUNNEL_STATUS_PAUSED:
		return "paused"
	default:
		return "stopped"
	}
}

func protoTunnelType(t borev1.TunnelType) string {
	switch t {
	case borev1.TunnelType_TUNNEL_TYPE_REMOTE:
		return "remote"
	case borev1.TunnelType_TUNNEL_TYPE_DYNAMIC:
		return "dynamic"
	case borev1.TunnelType_TUNNEL_TYPE_K8S:
		return "k8s"
	default:
		return "local"
	}
}
