package event

import (
	"sync"
	"testing"
	"time"
)

func TestPublishSubscribe(t *testing.T) {
	bus := NewBus()
	id, ch := bus.Subscribe(16)
	defer bus.Unsubscribe(id)

	bus.Publish(Event{Type: TunnelConnected, TunnelName: "test", Message: "hello"})

	select {
	case evt := <-ch:
		if evt.Type != TunnelConnected {
			t.Fatalf("expected TunnelConnected, got %v", evt.Type)
		}
		if evt.TunnelName != "test" {
			t.Fatalf("expected tunnel name 'test', got %q", evt.TunnelName)
		}
		if evt.Timestamp.IsZero() {
			t.Fatal("expected non-zero timestamp")
		}
	case <-time.After(time.Second):
		t.Fatal("timeout waiting for event")
	}
}

func TestMultipleSubscribers(t *testing.T) {
	bus := NewBus()
	id1, ch1 := bus.Subscribe(16)
	id2, ch2 := bus.Subscribe(16)
	defer bus.Unsubscribe(id1)
	defer bus.Unsubscribe(id2)

	bus.Publish(Event{Type: TunnelError, Message: "fail"})

	for _, ch := range []<-chan Event{ch1, ch2} {
		select {
		case evt := <-ch:
			if evt.Type != TunnelError {
				t.Fatalf("expected TunnelError, got %v", evt.Type)
			}
		case <-time.After(time.Second):
			t.Fatal("timeout")
		}
	}
}

func TestUnsubscribe(t *testing.T) {
	bus := NewBus()
	id, ch := bus.Subscribe(16)

	bus.Unsubscribe(id)

	// Channel should be closed after unsubscribe.
	_, ok := <-ch
	if ok {
		t.Fatal("expected channel to be closed")
	}

	// Publishing after unsubscribe should not panic.
	bus.Publish(Event{Type: ConfigReloaded})
}

func TestNonBlockingPublish(t *testing.T) {
	bus := NewBus()
	// Subscribe with buffer size 1.
	id, ch := bus.Subscribe(1)
	defer bus.Unsubscribe(id)

	// Fill the buffer.
	bus.Publish(Event{Type: TunnelConnected, Message: "1"})

	// This should not block — the event is dropped.
	bus.Publish(Event{Type: TunnelConnected, Message: "2"})
	bus.Publish(Event{Type: TunnelConnected, Message: "3"})

	// Only the first event should be in the channel.
	evt := <-ch
	if evt.Message != "1" {
		t.Fatalf("expected message '1', got %q", evt.Message)
	}

	// Channel should be empty now.
	select {
	case <-ch:
		t.Fatal("expected empty channel")
	default:
		// Good.
	}
}

func TestConcurrentPublish(t *testing.T) {
	bus := NewBus()
	id, ch := bus.Subscribe(1000)
	defer bus.Unsubscribe(id)

	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			bus.Publish(Event{Type: TunnelConnected})
		}()
	}
	wg.Wait()

	// Drain and count.
	count := 0
	for {
		select {
		case <-ch:
			count++
		default:
			goto done
		}
	}
done:
	if count != 100 {
		t.Fatalf("expected 100 events, got %d", count)
	}
}

func TestTimestampAutoSet(t *testing.T) {
	bus := NewBus()
	id, ch := bus.Subscribe(16)
	defer bus.Unsubscribe(id)

	before := time.Now()
	bus.Publish(Event{Type: TunnelDisconnected})
	after := time.Now()

	evt := <-ch
	if evt.Timestamp.Before(before) || evt.Timestamp.After(after) {
		t.Fatalf("timestamp %v not between %v and %v", evt.Timestamp, before, after)
	}
}

func TestCustomTimestampPreserved(t *testing.T) {
	bus := NewBus()
	id, ch := bus.Subscribe(16)
	defer bus.Unsubscribe(id)

	custom := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	bus.Publish(Event{Type: TunnelRetrying, Timestamp: custom})

	evt := <-ch
	if !evt.Timestamp.Equal(custom) {
		t.Fatalf("expected custom timestamp, got %v", evt.Timestamp)
	}
}
