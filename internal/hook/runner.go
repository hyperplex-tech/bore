package hook

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"time"

	"github.com/rs/zerolog/log"
)

const defaultTimeout = 30 * time.Second

// Env holds the environment context for hook execution.
type Env struct {
	TunnelName string
	Group      string
	LocalHost  string
	LocalPort  int
	RemoteHost string
	RemotePort int
	SSHHost    string
	SSHPort    int
	SSHUser    string
	Status     string // "connecting", "connected", "disconnecting", "disconnected"
}

// Run executes a hook command string in a shell with bore-specific environment
// variables. Returns an error if the command fails or times out.
func Run(command string, env Env) error {
	if command == "" {
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, "sh", "-c", command)
	cmd.Env = append(os.Environ(), envVars(env)...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	log.Debug().
		Str("tunnel", env.TunnelName).
		Str("command", command).
		Str("status", env.Status).
		Msg("hook: executing")

	if err := cmd.Run(); err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return fmt.Errorf("hook timed out after %s: %s", defaultTimeout, command)
		}
		return fmt.Errorf("hook failed: %w", err)
	}

	return nil
}

func envVars(env Env) []string {
	return []string{
		"BORE_TUNNEL_NAME=" + env.TunnelName,
		"BORE_GROUP=" + env.Group,
		"BORE_LOCAL_HOST=" + env.LocalHost,
		"BORE_LOCAL_PORT=" + strconv.Itoa(env.LocalPort),
		"BORE_REMOTE_HOST=" + env.RemoteHost,
		"BORE_REMOTE_PORT=" + strconv.Itoa(env.RemotePort),
		"BORE_SSH_HOST=" + env.SSHHost,
		"BORE_SSH_PORT=" + strconv.Itoa(env.SSHPort),
		"BORE_SSH_USER=" + env.SSHUser,
		"BORE_STATUS=" + env.Status,
	}
}
