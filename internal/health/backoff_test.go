package health

import (
	"testing"
	"time"
)

func TestBackoffExponentialGrowth(t *testing.T) {
	b := NewBackoff(60 * time.Second)

	prev := time.Duration(0)
	for i := 0; i < 6; i++ {
		d := b.Next()
		if i > 0 && d <= prev/2 {
			t.Fatalf("attempt %d: delay %v did not grow from %v", i, d, prev)
		}
		prev = d
	}
}

func TestBackoffCap(t *testing.T) {
	b := NewBackoff(5 * time.Second)

	// Run enough attempts to exceed the cap.
	for i := 0; i < 20; i++ {
		d := b.Next()
		// With 25% jitter, max possible is 5s * 1.25 = 6.25s.
		if d > 7*time.Second {
			t.Fatalf("delay %v exceeded cap with jitter", d)
		}
	}
}

func TestBackoffReset(t *testing.T) {
	b := NewBackoff(60 * time.Second)

	// Advance a few attempts.
	b.Next()
	b.Next()
	b.Next()
	if b.Attempt() != 3 {
		t.Fatalf("expected 3 attempts, got %d", b.Attempt())
	}

	b.Reset()
	if b.Attempt() != 0 {
		t.Fatalf("expected 0 after reset, got %d", b.Attempt())
	}

	// First delay after reset should be ~1s (±25% jitter).
	d := b.Next()
	if d < 500*time.Millisecond || d > 2*time.Second {
		t.Fatalf("first delay after reset out of range: %v", d)
	}
}

func TestBackoffNextDelaySecs(t *testing.T) {
	b := NewBackoff(60 * time.Second)
	secs := b.NextDelaySecs()
	if secs != 1 {
		t.Fatalf("expected 1s, got %d", secs)
	}
	// NextDelaySecs should not increment the counter.
	if b.Attempt() != 0 {
		t.Fatal("NextDelaySecs should not increment attempt counter")
	}
}
