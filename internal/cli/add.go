package cli

import (
	"fmt"

	borev1 "github.com/hyperplex-tech/bore/gen/bore/v1"
	internalconfig "github.com/hyperplex-tech/bore/internal/config"
	"github.com/spf13/cobra"
)

func newAddCmd() *cobra.Command {
	var (
		tunnelType string
		localHost  string
		localPort  int
		remoteHost string
		remotePort int
		sshHost    string
		sshPort    int
		sshUser    string
		group      string
		jumpHosts  []string
		authMethod string
		keyFile    string
		k8sContext   string
		k8sNamespace string
		k8sResource  string
		preConnect  string
		postConnect string
		reconnect   string
	)

	cmd := &cobra.Command{
		Use:   "add <name>",
		Short: "Add a new tunnel to the config",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]

			if tunnelType == "" {
				tunnelType = "local"
			}

			// Validate type.
			switch tunnelType {
			case "local", "remote", "dynamic", "k8s":
			default:
				return fmt.Errorf("invalid tunnel type %q (must be local, remote, dynamic, or k8s)", tunnelType)
			}

			if localPort == 0 {
				return fmt.Errorf("--local-port is required")
			}

			// Type-specific validation.
			switch tunnelType {
			case "k8s":
				if k8sResource == "" || remotePort == 0 {
					return fmt.Errorf("k8s tunnels require --k8s-resource and --remote-port")
				}
			case "dynamic":
				if sshHost == "" {
					return fmt.Errorf("dynamic (SOCKS5) tunnels require --via (SSH host)")
				}
			default: // local, remote
				if remoteHost == "" || remotePort == 0 {
					return fmt.Errorf("--remote-host and --remote-port are required")
				}
				if sshHost == "" {
					return fmt.Errorf("--via (SSH host) is required")
				}
			}

			if group == "" {
				group = "default"
			}

			tc := internalconfig.TunnelConfig{
				Name:         name,
				Type:         tunnelType,
				LocalHost:    localHost,
				LocalPort:    localPort,
				RemoteHost:   remoteHost,
				RemotePort:   remotePort,
				SSHHost:      sshHost,
				SSHPort:      sshPort,
				SSHUser:      sshUser,
				JumpHosts:    jumpHosts,
				AuthMethod:   authMethod,
				IdentityFile: keyFile,
				K8sContext:   k8sContext,
				K8sNamespace: k8sNamespace,
				K8sResource:  k8sResource,
			}

			// Hooks.
			if preConnect != "" || postConnect != "" {
				tc.Hooks = &internalconfig.Hooks{
					PreConnect:  preConnect,
					PostConnect: postConnect,
				}
			}

			// Reconnect.
			switch reconnect {
			case "yes", "true":
				b := true
				tc.Reconnect = &b
			case "no", "false":
				b := false
				tc.Reconnect = &b
			case "", "default":
				// leave nil — use global default
			default:
				return fmt.Errorf("invalid --reconnect value %q (must be yes, no, or default)", reconnect)
			}

			configPath := internalconfig.ConfigPath()
			if err := internalconfig.AddTunnel(configPath, group, tc); err != nil {
				return fmt.Errorf("adding tunnel: %w", err)
			}

			fmt.Printf("Added tunnel %q to group %q\n", name, group)

			// Print summary based on type.
			switch tunnelType {
			case "k8s":
				fmt.Printf("  :%d → %s:%d (k8s)\n", localPort, k8sResource, remotePort)
			case "dynamic":
				fmt.Printf("  SOCKS5 proxy on :%d via %s\n", localPort, sshHost)
			default:
				fmt.Printf("  %s:%d → %s:%d via %s\n", "localhost", localPort, remoteHost, remotePort, sshHost)
			}

			// Trigger config reload if daemon is running.
			if clients, err := Dial(socketPath); err == nil {
				defer clients.Close()
				clients.Daemon.ReloadConfig(cmd.Context(), &borev1.ReloadConfigRequest{})
				fmt.Println("Daemon config reloaded.")
			}

			return nil
		},
	}

	cmd.Flags().StringVar(&tunnelType, "type", "", "tunnel type: local, remote, dynamic, k8s (default: local)")
	cmd.Flags().StringVar(&localHost, "local-host", "", "local bind address (default: 127.0.0.1)")
	cmd.Flags().IntVar(&localPort, "local-port", 0, "local port to bind")
	cmd.Flags().StringVar(&remoteHost, "remote-host", "", "remote host to forward to")
	cmd.Flags().IntVar(&remotePort, "remote-port", 0, "remote port to forward to")
	cmd.Flags().StringVar(&sshHost, "via", "", "SSH bastion/jump host")
	cmd.Flags().IntVar(&sshPort, "ssh-port", 0, "SSH port (default from config)")
	cmd.Flags().StringVar(&sshUser, "ssh-user", "", "SSH username")
	cmd.Flags().StringVar(&group, "group", "", "group name (default: \"default\")")
	cmd.Flags().StringSliceVar(&jumpHosts, "jump", nil, "jump hosts (comma-separated)")
	cmd.Flags().StringVar(&authMethod, "auth", "", "auth method: agent, key")
	cmd.Flags().StringVar(&keyFile, "key", "", "path to SSH private key")
	cmd.Flags().StringVar(&k8sContext, "k8s-context", "", "Kubernetes context")
	cmd.Flags().StringVar(&k8sNamespace, "k8s-namespace", "", "Kubernetes namespace")
	cmd.Flags().StringVar(&k8sResource, "k8s-resource", "", "Kubernetes resource (e.g. svc/my-service)")
	cmd.Flags().StringVar(&preConnect, "pre-connect", "", "shell command to run before connecting")
	cmd.Flags().StringVar(&postConnect, "post-connect", "", "shell command to run after connecting")
	cmd.Flags().StringVar(&reconnect, "reconnect", "", "auto-reconnect: yes, no, or default")

	return cmd
}
