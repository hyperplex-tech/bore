package cli

import (
	"context"
	"fmt"
	"os"
	"text/tabwriter"
	"time"

	borev1 "github.com/hyperplex-tech/bore/gen/bore/v1"
	internalconfig "github.com/hyperplex-tech/bore/internal/config"
	"github.com/spf13/cobra"
)

func newGroupCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "group",
		Short: "Manage tunnel groups",
	}

	cmd.AddCommand(
		newGroupListCmd(),
		newGroupAddCmd(),
		newGroupRenameCmd(),
		newGroupDeleteCmd(),
	)

	return cmd
}

func newGroupListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List all groups",
		RunE: func(cmd *cobra.Command, args []string) error {
			clients, err := Dial(socketPath)
			if err != nil {
				return fmt.Errorf("cannot connect to daemon: %w", err)
			}
			defer clients.Close()

			ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
			defer cancel()

			resp, err := clients.Group.ListGroups(ctx, &borev1.ListGroupsRequest{})
			if err != nil {
				return fmt.Errorf("listing groups: %w", err)
			}

			if len(resp.Groups) == 0 {
				fmt.Println("No groups configured.")
				return nil
			}

			w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
			fmt.Fprintln(w, "NAME\tDESCRIPTION\tTUNNELS\tACTIVE")
			for _, g := range resp.Groups {
				fmt.Fprintf(w, "%s\t%s\t%d\t%d\n",
					g.Name, g.Description, g.TunnelCount, g.ActiveCount)
			}
			w.Flush()
			return nil
		},
	}
}

func newGroupAddCmd() *cobra.Command {
	var description string

	cmd := &cobra.Command{
		Use:   "add <name>",
		Short: "Create a new empty group",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]
			configPath := internalconfig.ConfigPath()

			if err := internalconfig.AddGroup(configPath, name, description); err != nil {
				return fmt.Errorf("adding group: %w", err)
			}

			fmt.Printf("Created group %q\n", name)

			// Trigger config reload if daemon is running.
			if clients, err := Dial(socketPath); err == nil {
				defer clients.Close()
				clients.Daemon.ReloadConfig(cmd.Context(), &borev1.ReloadConfigRequest{})
				fmt.Println("Daemon config reloaded.")
			}

			return nil
		},
	}

	cmd.Flags().StringVar(&description, "description", "", "group description")
	return cmd
}

func newGroupRenameCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "rename <old-name> <new-name>",
		Short: "Rename a group",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			oldName := args[0]
			newName := args[1]
			configPath := internalconfig.ConfigPath()

			if err := internalconfig.RenameGroup(configPath, oldName, newName); err != nil {
				return fmt.Errorf("renaming group: %w", err)
			}

			fmt.Printf("Renamed group %q to %q\n", oldName, newName)

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

func newGroupDeleteCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "delete <name>",
		Short: "Delete an empty group",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]
			configPath := internalconfig.ConfigPath()

			if err := internalconfig.RemoveGroup(configPath, name); err != nil {
				return fmt.Errorf("deleting group: %w", err)
			}

			fmt.Printf("Deleted group %q\n", name)

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
