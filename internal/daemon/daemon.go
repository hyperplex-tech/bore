package daemon

import (
	"context"
	"fmt"
	"os"
	"os/signal"

	"github.com/oklog/run"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"github.com/hyperplex-tech/bore/internal/config"
	"github.com/hyperplex-tech/bore/internal/engine"
	"github.com/hyperplex-tech/bore/internal/event"
	"github.com/hyperplex-tech/bore/internal/ipc"
	"github.com/hyperplex-tech/bore/internal/store"
	"github.com/hyperplex-tech/bore/internal/version"
)

// Daemon is the core bore daemon process.
type Daemon struct {
	cfg           *config.Config
	store         *store.Store
	bus           *event.Bus
	engine        *engine.Engine
	eventLogger   *store.EventLogger
	configWatcher *config.Watcher
	server        *Server
	lockFile      *os.File
	socketPath    string
	configPath    string
}

// Options configures the daemon.
type Options struct {
	ConfigPath string
	SocketPath string
	LogLevel   string
}

// New creates and initializes a new daemon.
func New(opts Options) (*Daemon, error) {
	// Set up logging.
	level, err := zerolog.ParseLevel(opts.LogLevel)
	if err != nil {
		level = zerolog.InfoLevel
	}
	zerolog.SetGlobalLevel(level)
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})

	// Ensure directories exist.
	if err := config.EnsureDirs(); err != nil {
		return nil, fmt.Errorf("creating directories: %w", err)
	}

	// Resolve paths.
	configPath := opts.ConfigPath
	if configPath == "" {
		configPath = config.ConfigPath()
	}
	socketPath := opts.SocketPath
	if socketPath == "" {
		socketPath = config.SocketPath()
	}

	// Acquire file lock.
	lockFile, err := acquireLock(config.LockPath())
	if err != nil {
		return nil, fmt.Errorf("acquiring lock: %w", err)
	}

	// Open store.
	st, err := store.New(config.DatabasePath())
	if err != nil {
		lockFile.Close()
		return nil, fmt.Errorf("opening store: %w", err)
	}

	// Reset stale tunnel states from a previous crash.
	if err := st.ResetTunnelStates(); err != nil {
		st.Close()
		lockFile.Close()
		return nil, fmt.Errorf("resetting tunnel states: %w", err)
	}

	// Load config.
	cfg, err := config.LoadOrDefault(configPath)
	if err != nil {
		st.Close()
		lockFile.Close()
		return nil, fmt.Errorf("loading config: %w", err)
	}

	bus := event.NewBus()
	eng := engine.NewEngine(bus)
	eng.LoadConfig(cfg)
	eventLogger := store.NewEventLogger(st, bus)

	d := &Daemon{
		cfg:         cfg,
		store:       st,
		bus:         bus,
		engine:      eng,
		eventLogger: eventLogger,
		lockFile:    lockFile,
		socketPath:  socketPath,
		configPath:  configPath,
	}

	// Create gRPC server.
	d.server = NewServer(d)

	return d, nil
}

// Run starts the daemon and blocks until shutdown.
func (d *Daemon) Run() error {
	log.Info().
		Str("version", version.Version).
		Str("socket", d.socketPath).
		Str("config", d.configPath).
		Int("tunnels", len(d.cfg.AllTunnels())).
		Msg("starting bore daemon")

	// Remove stale socket file.
	ipc.Cleanup(d.socketPath)

	lis, err := ipc.Listen(d.socketPath)
	if err != nil {
		return fmt.Errorf("listening on socket: %w", err)
	}

	// Watch config file for changes (auto-reload).
	if _, err := os.Stat(d.configPath); err == nil {
		cw, err := config.NewWatcher(d.configPath, d.ReloadConfig)
		if err != nil {
			log.Warn().Err(err).Msg("config file watcher failed to start")
		} else {
			d.configWatcher = cw
			log.Info().Str("file", d.configPath).Msg("watching config for changes")
		}
	}

	var g run.Group

	// Actor 1: gRPC server.
	g.Add(func() error {
		log.Info().Msg("gRPC server listening")
		return d.server.Serve(lis)
	}, func(error) {
		d.server.GracefulStop()
	})

	// Actor 2: Signal handler (interrupt/terminate).
	{
		ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
		g.Add(func() error {
			<-ctx.Done()
			log.Info().Msg("received shutdown signal")
			return nil
		}, func(error) {
			cancel()
		})
	}

	// Actor 3: Platform-specific signal handling (e.g. SIGHUP on Unix).
	addPlatformActors(&g, d)

	err = g.Run()

	// Cleanup.
	d.shutdown()
	return err
}

// ReloadConfig re-reads the YAML config.
func (d *Daemon) ReloadConfig() error {
	cfg, err := config.Load(d.configPath)
	if err != nil {
		return err
	}
	d.cfg = cfg
	added, removed, updated := d.engine.Reconcile(cfg)
	d.bus.Publish(event.Event{
		Type:    event.ConfigReloaded,
		Message: fmt.Sprintf("loaded %d tunnels (+%d -%d ~%d)", len(cfg.AllTunnels()), added, removed, updated),
	})
	return nil
}

func (d *Daemon) shutdown() {
	log.Info().Msg("shutting down")
	if d.configWatcher != nil {
		d.configWatcher.Close()
	}
	d.engine.Shutdown()
	d.eventLogger.Stop()
	os.Remove(d.socketPath)
	d.store.Close()
	releaseLock(d.lockFile)
}

