package cli

import (
	"fmt"
	"os"
	"os/exec"

	borev1 "github.com/hyperplex-tech/bore/gen/bore/v1"
	internalconfig "github.com/hyperplex-tech/bore/internal/config"
	"github.com/spf13/cobra"
)

func newEditCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "edit",
		Short: "Open the tunnel config in $EDITOR",
		RunE: func(cmd *cobra.Command, args []string) error {
			configPath := internalconfig.ConfigPath()

			// Ensure the config file exists.
			if _, err := os.Stat(configPath); os.IsNotExist(err) {
				// Create a default config.
				cfg := internalconfig.Defaults()
				if err := internalconfig.Save(configPath, &cfg); err != nil {
					return fmt.Errorf("creating default config: %w", err)
				}
			}

			editor := os.Getenv("EDITOR")
			if editor == "" {
				editor = os.Getenv("VISUAL")
			}
			if editor == "" {
				editor = "vi"
			}

			// Get the file's mtime before editing.
			infoBefore, _ := os.Stat(configPath)

			editorCmd := exec.Command(editor, configPath)
			editorCmd.Stdin = os.Stdin
			editorCmd.Stdout = os.Stdout
			editorCmd.Stderr = os.Stderr

			if err := editorCmd.Run(); err != nil {
				return fmt.Errorf("editor exited with error: %w", err)
			}

			// Check if the file was modified.
			infoAfter, _ := os.Stat(configPath)
			if infoBefore != nil && infoAfter != nil && infoBefore.ModTime().Equal(infoAfter.ModTime()) {
				fmt.Println("No changes detected.")
				return nil
			}

			// Validate the config.
			if _, err := internalconfig.Load(configPath); err != nil {
				return fmt.Errorf("config validation failed: %w", err)
			}

			fmt.Println("Config saved.")

			// Trigger config reload if daemon is running.
			if clients, err := Dial(socketPath); err == nil {
				defer clients.Close()
				resp, err := clients.Daemon.ReloadConfig(cmd.Context(), &borev1.ReloadConfigRequest{})
				if err == nil {
					fmt.Printf("Daemon reloaded: %d tunnels loaded.\n", resp.TunnelsLoaded)
				}
			}

			return nil
		},
	}
}
