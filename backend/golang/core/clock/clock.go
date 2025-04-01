// Package clock provides a time abstraction interface for testability and
// custom time implementations. The default implementation uses system time.
package clock

import (
	"errors"
	"time"
)

// realClock implements Clock using system time functions.
type realClock struct{}

// New creates a Clock instance using the host system's time.
func New() Clock {
	return &realClock{}
}

// After implements Clock interface for realClock.
func (c *realClock) After(d time.Duration) (<-chan time.Time, error) {
	if d <= 0 {
		return nil, errors.New("clock: duration must be positive")
	}
	return time.After(d), nil
}

// Now implements Clock interface for realClock.
func (c *realClock) Now() time.Time {
	return time.Now()
}

// Since implements Clock interface for realClock.
func (c *realClock) Since(t time.Time) time.Duration {
	return time.Since(t)
}

// Until implements Clock interface for realClock.
func (c *realClock) Until(t time.Time) time.Duration {
	return time.Until(t)
}

// Sleep implements Clock interface for realClock.
func (c *realClock) Sleep(d time.Duration) error {
	if d <= 0 {
		return errors.New("clock: duration must be positive")
	}
	time.Sleep(d)
	return nil
}

// Tick implements Clock interface for realClock.
func (c *realClock) Tick(d time.Duration) (<-chan time.Time, func(), error) {
	if d <= 0 {
		return nil, nil, errors.New("clock: duration must be positive")
	}

	ticker := time.NewTicker(d)
	stop := make(chan struct{})

	// Proxy channel to allow stopping
	ch := make(chan time.Time, 1)

	go func() {
		defer ticker.Stop()
		defer close(ch)
		for {
			select {
			case t := <-ticker.C:
				select {
				case ch <- t:
				default: // Skip if previous tick not consumed
				}
			case <-stop:
				return
			}
		}
	}()

	return ch, func() { close(stop) }, nil
}
