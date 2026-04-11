package cli

import (
	"context"
	"fmt"
	"os"
	"text/tabwriter"
	"time"

	borev1 "github.com/hyperplex-tech/bore/gen/bore/v1"
	"github.com/spf13/cobra"
)

func newStatusCmd() *cobra.Command {
	var (
		group        string
		statusFilter string
	)
	cmd := &cobra.Command{
		Use:   "status",
		Short: "Show tunnel status",
		RunE: func(cmd *cobra.Command, args []string) error {
			clients, err := Dial(socketPath)
			if err != nil {
				return fmt.Errorf("cannot connect to daemon: %w", err)
			}
			defer clients.Close()

			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			req := &borev1.ListRequest{Group: group}

			// Apply status filter if specified.
			if statusFilter != "" {
				sf, err := parseStatusFilter(statusFilter)
				if err != nil {
					return err
				}
				req.StatusFilter = sf
			}

			resp, err := clients.Tunnel.List(ctx, req)
			if err != nil {
				return fmt.Errorf("listing tunnels: %w", err)
			}

			if len(resp.Tunnels) == 0 {
				fmt.Println("No tunnels configured.")
				return nil
			}

			w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
			fmt.Fprintln(w, "NAME\tGROUP\tSTATUS\tLOCAL\tREMOTE\tVIA\tUPTIME")
			for _, t := range resp.Tunnels {
				local := fmt.Sprintf("%s:%d", t.LocalHost, t.LocalPort)
				remote := fmt.Sprintf("%s:%d", t.RemoteHost, t.RemotePort)
				status := statusString(t.Status)
				uptime := "-"

				if t.Status == borev1.TunnelStatus_TUNNEL_STATUS_ACTIVE && t.ConnectedAt != nil {
					uptime = formatDuration(time.Since(t.ConnectedAt.AsTime()))
				} else if t.Status == borev1.TunnelStatus_TUNNEL_STATUS_ERROR {
					status = fmt.Sprintf("error: %s", t.ErrorMessage)
				}

				fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\t%s\n",
					t.Name, t.Group, status, local, remote, t.SshHost, uptime)
			}
			w.Flush()
			return nil
		},
	}
	cmd.Flags().StringVar(&group, "group", "", "filter by group")
	cmd.Flags().StringVar(&statusFilter, "status", "", "filter by status: active, stopped, error, connecting, paused")
	return cmd
}

func parseStatusFilter(s string) (borev1.TunnelStatus, error) {
	switch s {
	case "active":
		return borev1.TunnelStatus_TUNNEL_STATUS_ACTIVE, nil
	case "stopped":
		return borev1.TunnelStatus_TUNNEL_STATUS_STOPPED, nil
	case "connecting":
		return borev1.TunnelStatus_TUNNEL_STATUS_CONNECTING, nil
	case "error":
		return borev1.TunnelStatus_TUNNEL_STATUS_ERROR, nil
	case "paused":
		return borev1.TunnelStatus_TUNNEL_STATUS_PAUSED, nil
	default:
		return 0, fmt.Errorf("invalid status filter %q (must be active, stopped, error, connecting, or paused)", s)
	}
}

func statusString(s borev1.TunnelStatus) string {
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

func formatDuration(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%ds", int(d.Seconds()))
	}
	if d < time.Hour {
		return fmt.Sprintf("%dm%ds", int(d.Minutes()), int(d.Seconds())%60)
	}
	return fmt.Sprintf("%dh%dm", int(d.Hours()), int(d.Minutes())%60)
}
