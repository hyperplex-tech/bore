package event

import (
	"sync"
	"time"
)

// Type represents a tunnel lifecycle event type.
type Type int

const (
	TunnelConnected Type = iota + 1
	TunnelDisconnected
	TunnelError
	TunnelRetrying
	ConfigReloaded
)

// Event is a tunnel lifecycle event.
type Event struct {
	Type       Type
	TunnelName string
	Message    string
	Timestamp  time.Time
}

// Bus is a simple in-process pub/sub event bus.
type Bus struct {
	mu   sync.RWMutex
	subs map[int]chan Event
	next int
}

// NewBus creates a new event bus.
func NewBus() *Bus {
	return &Bus{
		subs: make(map[int]chan Event),
	}
}

// Publish sends an event to all subscribers (non-blocking).
func (b *Bus) Publish(e Event) {
	if e.Timestamp.IsZero() {
		e.Timestamp = time.Now()
	}
	b.mu.RLock()
	defer b.mu.RUnlock()
	for _, ch := range b.subs {
		select {
		case ch <- e:
		default:
			// Drop if subscriber is slow — avoid blocking the publisher.
		}
	}
}

// Subscribe returns a channel that receives events. Call Unsubscribe with the
// returned ID when done.
func (b *Bus) Subscribe(bufSize int) (int, <-chan Event) {
	if bufSize <= 0 {
		bufSize = 64
	}
	ch := make(chan Event, bufSize)
	b.mu.Lock()
	id := b.next
	b.next++
	b.subs[id] = ch
	b.mu.Unlock()
	return id, ch
}

// Unsubscribe removes a subscriber and closes its channel.
func (b *Bus) Unsubscribe(id int) {
	b.mu.Lock()
	if ch, ok := b.subs[id]; ok {
		delete(b.subs, id)
		close(ch)
	}
	b.mu.Unlock()
}
