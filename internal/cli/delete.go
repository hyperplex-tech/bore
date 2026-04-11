package cli

import (
	"fmt"

	borev1 "github.com/hyperplex-tech/bore/gen/bore/v1"
	internalconfig "github.com/hyperplex-tech/bore/internal/config"
	"github.com/spf13/cobra"
)

func newDeleteCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "delete <name>",
		Short: "Delete a tunnel from the config",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]
			configPath := internalconfig.ConfigPath()

			if err := internalconfig.RemoveTunnel(configPath, name); err != nil {
				return fmt.Errorf("deleting tunnel: %w", err)
			}

			fmt.Printf("Deleted tunnel %q\n", name)

			// Trigger config reload if daemon is running.
			if clients, err := Dial(socketPath); err == nil {
				defer clients.Close()
				clients.Daemon.ReloadConfig(cmd.Context(), &borev1.ReloadConfigRequest{})
				fmt.Println("Daemon config reloaded.")
			}

			return nil
		},
	}
}
