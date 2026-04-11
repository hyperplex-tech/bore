package config

import (
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/rs/zerolog/log"
)

// Watcher monitors the config file for changes and calls the reload function.
type Watcher struct {
	watcher  *fsnotify.Watcher
	path     string
	reloadFn func() error
	done     chan struct{}
}

// NewWatcher creates a file watcher that triggers reloadFn when the config
// file is modified. Uses a 500ms debounce to coalesce rapid writes (e.g.
// from editors that write-then-rename).
func NewWatcher(path string, reloadFn func() error) (*Watcher, error) {
	w, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}

	if err := w.Add(path); err != nil {
		w.Close()
		return nil, err
	}

	cw := &Watcher{
		watcher:  w,
		path:     path,
		reloadFn: reloadFn,
		done:     make(chan struct{}),
	}

	go cw.run()
	return cw, nil
}

// Close stops the watcher.
func (cw *Watcher) Close() {
	cw.watcher.Close()
	<-cw.done
}

func (cw *Watcher) run() {
	defer close(cw.done)

	var debounce *time.Timer

	for {
		select {
		case event, ok := <-cw.watcher.Events:
			if !ok {
				return
			}
			if event.Has(fsnotify.Write) || event.Has(fsnotify.Create) {
				// Debounce: wait 500ms for the last write before reloading.
				if debounce != nil {
					debounce.Stop()
				}
				debounce = time.AfterFunc(500*time.Millisecond, func() {
					log.Info().Str("file", cw.path).Msg("config file changed, reloading")
					if err := cw.reloadFn(); err != nil {
						log.Error().Err(err).Msg("config reload after file change failed")
					}
				})
			}
			if event.Has(fsnotify.Remove) || event.Has(fsnotify.Rename) {
				// Some editors (vim) delete+rename. Re-add the watch.
				cw.watcher.Add(cw.path)
			}
		case err, ok := <-cw.watcher.Errors:
			if !ok {
				return
			}
			log.Warn().Err(err).Msg("config watcher error")
		}
	}
}
