package cli

import (
	"context"
	"fmt"
	"time"

	borev1 "github.com/hyperplex-tech/bore/gen/bore/v1"
	"github.com/spf13/cobra"
)

func newDownCmd() *cobra.Command {
	var tunnelNames []string

	cmd := &cobra.Command{
		Use:   "down [group]",
		Short: "Disconnect tunnels (by group, by name, or all)",
		Long: `Disconnect tunnels. With no arguments, disconnects all tunnels.
With a positional argument, treats it as a group name.
Use -t/--tunnel to disconnect specific tunnels by name.`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clients, err := Dial(socketPath)
			if err != nil {
				return fmt.Errorf("cannot connect to daemon: %w", err)
			}
			defer clients.Close()

			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()

			req := &borev1.DisconnectRequest{}

			if len(tunnelNames) > 0 {
				req.Names = tunnelNames
			} else if len(args) == 1 {
				req.Group = args[0]
			}

			_, err = clients.Tunnel.Disconnect(ctx, req)
			if err != nil {
				return fmt.Errorf("disconnecting tunnels: %w", err)
			}

			if len(tunnelNames) > 0 {
				fmt.Printf("Disconnected tunnels: %v\n", tunnelNames)
			} else if len(args) == 1 {
				fmt.Printf("Disconnected %s tunnels.\n", args[0])
			} else {
				fmt.Println("Disconnected all tunnels.")
			}
			return nil
		},
	}

	cmd.Flags().StringSliceVarP(&tunnelNames, "tunnel", "t", nil, "tunnel name(s) to disconnect (can be repeated)")
	return cmd
}
