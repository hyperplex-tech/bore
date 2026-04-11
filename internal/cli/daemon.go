package cli

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"time"

	borev1 "github.com/hyperplex-tech/bore/gen/bore/v1"
	"github.com/hyperplex-tech/bore/internal/config"
	"github.com/spf13/cobra"
)

func newDaemonCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "daemon",
		Short: "Manage the bore daemon",
	}

	cmd.AddCommand(
		newDaemonStartCmd(),
		newDaemonStopCmd(),
		newDaemonStatusCmd(),
	)

	return cmd
}

func newDaemonStartCmd() *cobra.Command {
	var foreground bool
	cmd := &cobra.Command{
		Use:   "start",
		Short: "Start the bore daemon",
		RunE: func(cmd *cobra.Command, args []string) error {
			if foreground {
				// Run in foreground — just exec the daemon binary inline.
				fmt.Println("Starting bore daemon in foreground...")
				daemon := exec.Command(os.Args[0]+"d") // "bored" binary
				daemon.Stdout = os.Stdout
				daemon.Stderr = os.Stderr
				daemon.Env = os.Environ()
				return daemon.Run()
			}

			// Background mode: start bored as a detached process.
			boredPath, err := exec.LookPath("bored")
			if err != nil {
				// Try looking next to the bore binary.
				boredPath = os.Args[0] + "d"
			}

			daemon := exec.Command(boredPath)
			daemon.Env = os.Environ()
			if err := daemon.Start(); err != nil {
				return fmt.Errorf("starting daemon: %w", err)
			}

			// Wait briefly and verify it's running.
			time.Sleep(500 * time.Millisecond)

			clients, err := Dial(socketPath)
			if err != nil {
				return fmt.Errorf("daemon started but cannot connect: %w", err)
			}
			defer clients.Close()

			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			status, err := clients.Daemon.Status(ctx, &borev1.StatusRequest{})
			if err != nil {
				return fmt.Errorf("daemon started but not responding: %w", err)
			}

			fmt.Printf("bore daemon v%s started (pid %d)\n", status.Version, daemon.Process.Pid)
			fmt.Printf("  socket: %s\n", status.SocketPath)
			fmt.Printf("  config: %s\n", status.ConfigPath)
			fmt.Printf("  tunnels: %d\n", status.TotalTunnels)
			return nil
		},
	}
	cmd.Flags().BoolVar(&foreground, "foreground", false, "run daemon in the foreground")
	return cmd
}

func newDaemonStopCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "stop",
		Short: "Stop the bore daemon",
		RunE: func(cmd *cobra.Command, args []string) error {
			clients, err := Dial(socketPath)
			if err != nil {
				return fmt.Errorf("cannot connect to daemon (is it running?): %w", err)
			}
			defer clients.Close()

			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			_, err = clients.Daemon.Shutdown(ctx, &borev1.ShutdownRequest{})
			if err != nil {
				return fmt.Errorf("shutdown request failed: %w", err)
			}

			fmt.Println("bore daemon stopped")
			return nil
		},
	}
}

func newDaemonStatusCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Show bore daemon status",
		RunE: func(cmd *cobra.Command, args []string) error {
			clients, err := Dial(socketPath)
			if err != nil {
				// Check if socket exists to give a better error.
				if _, statErr := os.Stat(config.SocketPath()); os.IsNotExist(statErr) {
					fmt.Println("bore daemon is not running")
					return nil
				}
				return fmt.Errorf("cannot connect to daemon: %w", err)
			}
			defer clients.Close()

			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			status, err := clients.Daemon.Status(ctx, &borev1.StatusRequest{})
			if err != nil {
				return fmt.Errorf("status request failed: %w", err)
			}

			fmt.Printf("bore daemon v%s\n", status.Version)
			fmt.Printf("  socket:    %s\n", status.SocketPath)
			fmt.Printf("  config:    %s\n", status.ConfigPath)
			fmt.Printf("  tunnels:   %d total, %d active\n", status.TotalTunnels, status.ActiveTunnels)
			if status.SshAgentAvailable {
				fmt.Printf("  ssh-agent: %d keys loaded\n", status.SshAgentKeys)
			} else {
				fmt.Printf("  ssh-agent: not available\n")
			}
			if status.TailscaleAvailable {
				if status.TailscaleConnected {
					fmt.Printf("  tailscale: connected (%s)\n", status.TailscaleIp)
				} else {
					fmt.Printf("  tailscale: installed but not connected\n")
				}
			}
			return nil
		},
	}
}
