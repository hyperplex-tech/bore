//go:build !windows

package daemon

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/oklog/run"
	"github.com/rs/zerolog/log"
)

// addPlatformActors adds SIGHUP config reload on Unix.
func addPlatformActors(g *run.Group, d *Daemon) {
	sighup := make(chan os.Signal, 1)
	signal.Notify(sighup, syscall.SIGHUP)
	ctx, cancel := context.WithCancel(context.Background())
	g.Add(func() error {
		for {
			select {
			case <-ctx.Done():
				return nil
			case <-sighup:
				if err := d.ReloadConfig(); err != nil {
					log.Error().Err(err).Msg("config reload failed")
				} else {
					log.Info().Msg("config reloaded")
				}
			}
		}
	}, func(error) {
		cancel()
	})
}
