package health

import (
	"math"
	"math/rand/v2"
	"time"
)

// Backoff computes jittered exponential backoff delays.
type Backoff struct {
	Base    time.Duration // initial delay (default 1s)
	Max     time.Duration // maximum delay cap (default 60s)
	Jitter  float64       // ±jitter fraction (default 0.25 = 25%)
	attempt int
}

// NewBackoff creates a backoff with sensible defaults.
func NewBackoff(maxInterval time.Duration) *Backoff {
	if maxInterval <= 0 {
		maxInterval = 60 * time.Second
	}
	return &Backoff{
		Base:   1 * time.Second,
		Max:    maxInterval,
		Jitter: 0.25,
	}
}

// Next returns the next backoff delay and increments the attempt counter.
// Sequence: 1s, 2s, 4s, 8s, 16s, 32s, 60s (capped), each ±25% jitter.
func (b *Backoff) Next() time.Duration {
	delay := float64(b.Base) * math.Pow(2, float64(b.attempt))
	if delay > float64(b.Max) {
		delay = float64(b.Max)
	}

	// Apply jitter: delay * (1 ± jitter).
	jitterRange := delay * b.Jitter
	delay = delay - jitterRange + rand.Float64()*2*jitterRange

	b.attempt++
	return time.Duration(delay)
}

// Reset resets the attempt counter (call after a successful connection).
func (b *Backoff) Reset() {
	b.attempt = 0
}

// Attempt returns the current attempt number.
func (b *Backoff) Attempt() int {
	return b.attempt
}

// NextDelaySecs returns the next delay in seconds (for display purposes)
// without incrementing the counter.
func (b *Backoff) NextDelaySecs() int {
	delay := float64(b.Base) * math.Pow(2, float64(b.attempt))
	if delay > float64(b.Max) {
		delay = float64(b.Max)
	}
	return int(time.Duration(delay).Seconds())
}
