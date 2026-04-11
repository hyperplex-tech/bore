package cli

import (
	"fmt"

	borev1 "github.com/hyperplex-tech/bore/gen/bore/v1"
	internalconfig "github.com/hyperplex-tech/bore/internal/config"
	"github.com/hyperplex-tech/bore/internal/profile"
	"github.com/spf13/cobra"
)

func newImportCmd() *cobra.Command {
	var (
		sshConfigPath string
		group         string
		dryRun        bool
	)

	cmd := &cobra.Command{
		Use:   "import",
		Short: "Import tunnels from SSH config",
		RunE: func(cmd *cobra.Command, args []string) error {
			hosts, err := profile.ImportSSHConfig(sshConfigPath)
			if err != nil {
				return fmt.Errorf("parsing SSH config: %w", err)
			}

			if len(hosts) == 0 {
				fmt.Println("No hosts with LocalForward directives found.")
				return nil
			}

			tunnels := profile.ToTunnelConfigs(hosts)
			fmt.Printf("Found %d tunnel(s) from %d SSH host(s):\n\n", len(tunnels), len(hosts))

			for _, tc := range tunnels {
				via := tc.SSHHost
				if tc.SSHUser != "" {
					via = tc.SSHUser + "@" + via
				}
				if len(tc.JumpHosts) > 0 {
					via += fmt.Sprintf(" (via %s)", tc.JumpHosts[0])
				}
				fmt.Printf("  %-20s %s:%d → %s:%d via %s\n",
					tc.Name, tc.LocalHost, tc.LocalPort,
					tc.RemoteHost, tc.RemotePort, via)
			}

			if dryRun {
				fmt.Println("\nDry run — no changes written.")
				return nil
			}

			if group == "" {
				group = "imported"
			}

			configPath := internalconfig.ConfigPath()
			added := 0
			for _, tc := range tunnels {
				if err := internalconfig.AddTunnel(configPath, group, tc); err != nil {
					fmt.Printf("  skip %s: %v\n", tc.Name, err)
				} else {
					added++
				}
			}

			fmt.Printf("\nImported %d tunnel(s) into group %q.\n", added, group)

			// Trigger config reload if daemon is running.
			if clients, err := Dial(socketPath); err == nil {
				defer clients.Close()
				clients.Daemon.ReloadConfig(cmd.Context(), &borev1.ReloadConfigRequest{})
				fmt.Println("Daemon config reloaded.")
			}

			return nil
		},
	}

	cmd.Flags().StringVar(&sshConfigPath, "ssh-config", "", "path to SSH config (default: ~/.ssh/config)")
	cmd.Flags().StringVar(&group, "group", "", "target group name (default: \"imported\")")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "show what would be imported without writing")

	return cmd
}
