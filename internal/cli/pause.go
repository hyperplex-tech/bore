package cli

import (
	"context"
	"fmt"
	"time"

	borev1 "github.com/hyperplex-tech/bore/gen/bore/v1"
	"github.com/spf13/cobra"
)

func newPauseCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "pause <name>",
		Short: "Pause (disconnect) a single tunnel",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]

			clients, err := Dial(socketPath)
			if err != nil {
				return fmt.Errorf("cannot connect to daemon: %w", err)
			}
			defer clients.Close()

			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()

			resp, err := clients.Tunnel.Pause(ctx, &borev1.PauseRequest{Name: name})
			if err != nil {
				return fmt.Errorf("pausing tunnel: %w", err)
			}

			fmt.Printf("Paused tunnel %q [%s]\n", resp.Tunnel.Name, statusString(resp.Tunnel.Status))
			return nil
		},
	}
}
