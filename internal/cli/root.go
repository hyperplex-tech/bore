package cli

import (
	"github.com/spf13/cobra"
)

var socketPath string

// NewRootCmd creates the root bore CLI command.
func NewRootCmd() *cobra.Command {
	root := &cobra.Command{
		Use:   "bore",
		Short: "SSH tunnel manager",
		Long:  "Bore — a modern SSH tunnel manager with daemon, CLI, and GUI.",
	}

	root.PersistentFlags().StringVar(&socketPath, "socket", "", "daemon socket path (default: $XDG_DATA_HOME/bore/bored.sock)")

	root.AddCommand(
		newDaemonCmd(),
		newStatusCmd(),
		newUpCmd(),
		newDownCmd(),
		newLogsCmd(),
		newAddCmd(),
		newDeleteCmd(),
		newRetryCmd(),
		newPauseCmd(),
		newGroupCmd(),
		newImportCmd(),
		newEditCmd(),
	)

	return root
}
