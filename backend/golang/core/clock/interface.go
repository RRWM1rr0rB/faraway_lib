package clock

import "time"

// Clock defines operations for time management, allowing substitution
// of real-time implementations with mock clocks for testing.
type Clock interface {
	// After waits for duration d and sends current time on returned channel.
	// Returns error for non-positive durations.
	After(d time.Duration) (<-chan time.Time, error)

	// Now returns current time according to the clock's time source.
	Now() time.Time

	// Since calculates duration elapsed since time t.
	Since(t time.Time) time.Duration

	// Until calculates duration remaining until time t.
	Until(t time.Time) time.Duration

	// Sleep blocks the current goroutine for duration d.
	// Returns error for non-positive durations.
	Sleep(d time.Duration) error

	// Tick generates time events at interval d. Returns a receive-only channel
	// for time events and a stop function to release resources.
	// The stop function must be called to prevent resource leaks.
	// Returns error for non-positive durations.
	Tick(d time.Duration) (<-chan time.Time, func(), error)
}
