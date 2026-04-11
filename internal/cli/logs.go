package cli

import (
	"context"
	"fmt"
	"io"
	"time"

	borev1 "github.com/hyperplex-tech/bore/gen/bore/v1"
	"github.com/spf13/cobra"
)

func newLogsCmd() *cobra.Command {
	var (
		tail   int32
		follow bool
	)

	cmd := &cobra.Command{
		Use:   "logs [tunnel-name]",
		Short: "Show tunnel connection logs",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clients, err := Dial(socketPath)
			if err != nil {
				return fmt.Errorf("cannot connect to daemon: %w", err)
			}
			defer clients.Close()

			ctx := context.Background()
			if !follow {
				var cancel context.CancelFunc
				ctx, cancel = context.WithTimeout(ctx, 10*time.Second)
				defer cancel()
			}

			req := &borev1.GetLogsRequest{
				Tail:   tail,
				Follow: follow,
			}
			if len(args) > 0 {
				req.Name = args[0]
			}

			stream, err := clients.Tunnel.GetLogs(ctx, req)
			if err != nil {
				return fmt.Errorf("streaming logs: %w", err)
			}

			for {
				entry, err := stream.Recv()
				if err == io.EOF {
					return nil
				}
				if err != nil {
					// Context cancelled (e.g. Ctrl-C) is normal for follow mode.
					if ctx.Err() != nil {
						return nil
					}
					return fmt.Errorf("receiving log: %w", err)
				}

				ts := entry.Timestamp.AsTime().Local().Format("15:04:05")
				fmt.Printf("%s [%s] %s\n", ts, entry.Level, entry.Message)
			}
		},
	}

	cmd.Flags().Int32VarP(&tail, "tail", "n", 50, "number of recent entries to show")
	cmd.Flags().BoolVarP(&follow, "follow", "f", false, "stream live log entries")
	return cmd
}
