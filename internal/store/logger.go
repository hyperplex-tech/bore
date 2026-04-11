package store

import (
	"github.com/rs/zerolog/log"

	"github.com/hyperplex-tech/bore/internal/event"
)

// EventLogger subscribes to the event bus and persists tunnel events to SQLite.
type EventLogger struct {
	store *Store
	bus   *event.Bus
	subID int
	done  chan struct{}
}

// NewEventLogger creates and starts an event logger.
func NewEventLogger(store *Store, bus *event.Bus) *EventLogger {
	id, ch := bus.Subscribe(256)
	el := &EventLogger{
		store: store,
		bus:   bus,
		subID: id,
		done:  make(chan struct{}),
	}
	go el.run(ch)
	return el
}

// Stop unsubscribes from the event bus and waits for the logger to finish.
func (el *EventLogger) Stop() {
	el.bus.Unsubscribe(el.subID)
	<-el.done
}

func (el *EventLogger) run(ch <-chan event.Event) {
	defer close(el.done)

	for evt := range ch {
		tunnelName := evt.TunnelName
		if tunnelName == "" {
			// System events (e.g. config reload) have no tunnel —
			// skip DB logging since connection_log has a FK to tunnel_state.
			continue
		}

		level := eventLevel(evt.Type)
		msg := evt.Message
		if msg == "" {
			msg = eventDescription(evt.Type)
		}

		// Ensure the tunnel exists in tunnel_state before logging.
		_ = el.store.UpsertTunnelState(tunnelName, "", eventStatus(evt.Type), 0, "", "")

		if err := el.store.AppendLog(tunnelName, level, msg); err != nil {
			log.Warn().Err(err).
				Str("tunnel", tunnelName).
				Str("event", eventDescription(evt.Type)).
				Msg("failed to persist log entry")
		}
	}
}

func eventLevel(t event.Type) string {
	switch t {
	case event.TunnelConnected:
		return "info"
	case event.TunnelDisconnected:
		return "info"
	case event.TunnelError:
		return "error"
	case event.TunnelRetrying:
		return "warn"
	case event.ConfigReloaded:
		return "info"
	default:
		return "info"
	}
}

func eventStatus(t event.Type) string {
	switch t {
	case event.TunnelConnected:
		return "active"
	case event.TunnelDisconnected:
		return "stopped"
	case event.TunnelError:
		return "error"
	case event.TunnelRetrying:
		return "retrying"
	default:
		return "stopped"
	}
}

func eventDescription(t event.Type) string {
	switch t {
	case event.TunnelConnected:
		return "tunnel connected"
	case event.TunnelDisconnected:
		return "tunnel disconnected"
	case event.TunnelError:
		return "tunnel error"
	case event.TunnelRetrying:
		return "tunnel retrying"
	case event.ConfigReloaded:
		return "config reloaded"
	default:
		return "unknown event"
	}
}
