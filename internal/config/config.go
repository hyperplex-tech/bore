package config

import (
	"fmt"
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

// Config is the top-level bore configuration.
type Config struct {
	Version  int              `yaml:"version"`
	Defaults DefaultsConfig   `yaml:"defaults"`
	Groups   map[string]Group `yaml:"groups"`
}

// DefaultsConfig holds default values inherited by all tunnels.
type DefaultsConfig struct {
	SSHPort              int           `yaml:"ssh_port"`
	SSHUser              string        `yaml:"ssh_user"`
	AuthMethod           string        `yaml:"auth_method"`
	IdentityFile         string        `yaml:"identity_file"`
	Reconnect            bool          `yaml:"reconnect"`
	ReconnectMaxInterval time.Duration `yaml:"reconnect_max_interval"`
	KeepaliveInterval    time.Duration `yaml:"keepalive_interval"`
	KeepaliveMaxFailures int           `yaml:"keepalive_max_failures"`
}

// Group is a named collection of tunnels.
type Group struct {
	Description string         `yaml:"description"`
	Tunnels     []TunnelConfig `yaml:"tunnels"`
}

// TunnelConfig defines a single tunnel in YAML.
type TunnelConfig struct {
	Name         string   `yaml:"name"`
	Type         string   `yaml:"type,omitempty"`
	LocalHost    string   `yaml:"local_host,omitempty"`
	LocalPort    int      `yaml:"local_port"`
	RemoteHost   string   `yaml:"remote_host,omitempty"`
	RemotePort   int      `yaml:"remote_port"`
	SSHHost      string   `yaml:"ssh_host,omitempty"`
	SSHPort      int      `yaml:"ssh_port,omitempty"`
	SSHUser      string   `yaml:"ssh_user,omitempty"`
	AuthMethod   string   `yaml:"auth_method,omitempty"`
	IdentityFile string   `yaml:"identity_file,omitempty"`
	JumpHosts    []string `yaml:"jump_hosts,omitempty"`
	Via          string   `yaml:"via,omitempty"`
	K8sContext   string   `yaml:"k8s_context,omitempty"`
	K8sNamespace string   `yaml:"k8s_namespace,omitempty"`
	K8sResource  string   `yaml:"k8s_resource,omitempty"`
	Hooks        *Hooks   `yaml:"hooks,omitempty"`
	Reconnect    *bool    `yaml:"reconnect,omitempty"`
}

// Hooks defines pre/post-connect commands.
type Hooks struct {
	PreConnect  string `yaml:"pre_connect,omitempty"`
	PostConnect string `yaml:"post_connect,omitempty"`
}

// Defaults returns a Config with sensible default values.
func Defaults() Config {
	return Config{
		Version: 1,
		Defaults: DefaultsConfig{
			SSHPort:              22,
			AuthMethod:           "agent",
			Reconnect:            true,
			ReconnectMaxInterval: 60 * time.Second,
			KeepaliveInterval:    30 * time.Second,
			KeepaliveMaxFailures: 3,
		},
		Groups: make(map[string]Group),
	}
}

// Load reads and parses the YAML config file.
func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading config: %w", err)
	}

	cfg := Defaults()
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parsing config: %w", err)
	}

	cfg.applyDefaults()
	return &cfg, nil
}

// LoadOrDefault tries to load the config; returns defaults if the file doesn't exist.
func LoadOrDefault(path string) (*Config, error) {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		cfg := Defaults()
		return &cfg, nil
	}
	return Load(path)
}

// applyDefaults fills in tunnel-level fields from the top-level defaults.
func (c *Config) applyDefaults() {
	for groupName, group := range c.Groups {
		for i := range group.Tunnels {
			t := &group.Tunnels[i]
			if t.LocalHost == "" {
				t.LocalHost = "127.0.0.1"
			}
			if t.Type == "" {
				t.Type = "local"
			}
			if t.SSHPort == 0 {
				t.SSHPort = c.Defaults.SSHPort
			}
			if t.SSHUser == "" {
				t.SSHUser = c.Defaults.SSHUser
			}
			if t.AuthMethod == "" {
				t.AuthMethod = c.Defaults.AuthMethod
			}
			if t.IdentityFile == "" {
				t.IdentityFile = c.Defaults.IdentityFile
			}
			if t.Reconnect == nil {
				r := c.Defaults.Reconnect
				t.Reconnect = &r
			}
		}
		c.Groups[groupName] = group
	}
}

// ResolvedTunnel pairs a TunnelConfig with its group name and resolved
// defaults for health/reconnect settings.
type ResolvedTunnel struct {
	TunnelConfig
	Group                string
	ReconnectMaxInterval time.Duration
	KeepaliveInterval    time.Duration
	KeepaliveMaxFailures int
}

// AllTunnels returns a flat list of all tunnel configs with their group names.
func (c *Config) AllTunnels() []ResolvedTunnel {
	var tunnels []ResolvedTunnel
	for groupName, group := range c.Groups {
		for _, t := range group.Tunnels {
			tunnels = append(tunnels, c.resolve(t, groupName))
		}
	}
	return tunnels
}

// TunnelsByGroup returns tunnels belonging to a specific group.
func (c *Config) TunnelsByGroup(group string) ([]ResolvedTunnel, bool) {
	g, ok := c.Groups[group]
	if !ok {
		return nil, false
	}
	tunnels := make([]ResolvedTunnel, len(g.Tunnels))
	for i, t := range g.Tunnels {
		tunnels[i] = c.resolve(t, group)
	}
	return tunnels, true
}

// FindTunnel finds a tunnel by name across all groups.
func (c *Config) FindTunnel(name string) (*ResolvedTunnel, bool) {
	for groupName, group := range c.Groups {
		for _, t := range group.Tunnels {
			if t.Name == name {
				rt := c.resolve(t, groupName)
				return &rt, true
			}
		}
	}
	return nil, false
}

// resolve creates a ResolvedTunnel with health/reconnect fields from defaults.
func (c *Config) resolve(t TunnelConfig, group string) ResolvedTunnel {
	return ResolvedTunnel{
		TunnelConfig:         t,
		Group:                group,
		ReconnectMaxInterval: c.Defaults.ReconnectMaxInterval,
		KeepaliveInterval:    c.Defaults.KeepaliveInterval,
		KeepaliveMaxFailures: c.Defaults.KeepaliveMaxFailures,
	}
}
