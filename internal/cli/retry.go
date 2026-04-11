package cli

import (
	"context"
	"fmt"
	"time"

	borev1 "github.com/hyperplex-tech/bore/gen/bore/v1"
	"github.com/spf13/cobra"
)

func newRetryCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "retry <name>",
		Short: "Retry a failed tunnel connection",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]

			clients, err := Dial(socketPath)
			if err != nil {
				return fmt.Errorf("cannot connect to daemon: %w", err)
			}
			defer clients.Close()

			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()

			resp, err := clients.Tunnel.Retry(ctx, &borev1.RetryRequest{Name: name})
			if err != nil {
				return fmt.Errorf("retrying tunnel: %w", err)
			}

			fmt.Printf("Retrying tunnel %q [%s]\n", resp.Tunnel.Name, statusString(resp.Tunnel.Status))
			return nil
		},
	}
}
