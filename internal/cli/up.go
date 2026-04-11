package cli

import (
	"context"
	"fmt"
	"time"

	borev1 "github.com/hyperplex-tech/bore/gen/bore/v1"
	"github.com/spf13/cobra"
)

func newUpCmd() *cobra.Command {
	var tunnelNames []string

	cmd := &cobra.Command{
		Use:   "up [group]",
		Short: "Connect tunnels (by group, by name, or all)",
		Long: `Connect tunnels. With no arguments, connects all tunnels.
With a positional argument, treats it as a group name.
Use -t/--tunnel to connect specific tunnels by name.`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clients, err := Dial(socketPath)
			if err != nil {
				return fmt.Errorf("cannot connect to daemon: %w", err)
			}
			defer clients.Close()

			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()

			req := &borev1.ConnectRequest{}

			if len(tunnelNames) > 0 {
				req.Names = tunnelNames
			} else if len(args) == 1 {
				req.Group = args[0]
			}

			resp, err := clients.Tunnel.Connect(ctx, req)
			if err != nil {
				return fmt.Errorf("connecting tunnels: %w", err)
			}

			if len(resp.Tunnels) == 0 {
				fmt.Println("No tunnels to connect.")
			}
			for _, t := range resp.Tunnels {
				fmt.Printf("  %s → %s:%d via %s [%s]\n",
					t.Name, t.RemoteHost, t.RemotePort, t.SshHost, statusString(t.Status))
			}
			return nil
		},
	}

	cmd.Flags().StringSliceVarP(&tunnelNames, "tunnel", "t", nil, "tunnel name(s) to connect (can be repeated)")
	return cmd
}
