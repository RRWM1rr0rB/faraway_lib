// Package repeat provides configurable retry logic with support for
// exponential backoff, jitter, and custom retry policies.
package repeat

import (
	"context"
	"errors"
	"fmt"
	"math/rand/v2"
	"time"
)

// Operation defines a function signature for operations to retry.
// ctx: Context for cancellation and deadlines
// retryCount: Current attempt number (0-indexed)
// Returns: error if operation failed, nil on success
type Operation func(ctx context.Context, retryCount int) error

// Config holds retry configuration parameters.
type Config struct {
	MinTimeWait  time.Duration                        // Minimum delay between attempts
	MaxTimeWait  time.Duration                        // Maximum delay cap
	MaxRetries   int                                  // Maximum attempts (-1 = infinite)
	BackoffBase  time.Duration                        // Base duration for exponential backoff
	BackoffMax   time.Duration                        // Maximum backoff duration
	UseBackoff   bool                                 // Enable exponential backoff
	ErrorHandler func(err error) bool                 // Basic error filter
	JitterFunc   func(d time.Duration) time.Duration  // Delay randomizer
	ShouldRetry  func(retryCount int, err error) bool // Advanced retry policy
}

// Default configuration constants.
const (
	DefaultMinWait    = time.Second
	DefaultMaxWait    = time.Minute
	DefaultMaxRetries = -1
	DefaultBackoff    = 500 * time.Millisecond
	MaxBackoff        = 30 * time.Second
	MaxTotalDuration  = 24 * time.Hour
)

// OptionSetter modifies Config through functional options.
type OptionSetter func(*Config)

// WithMinWait sets minimum delay between attempts.
func WithMinWait(d time.Duration) OptionSetter {
	return func(c *Config) {
		if d >= 0 {
			c.MinTimeWait = d
		}
	}
}

// WithMaxWait sets maximum delay cap.
func WithMaxWait(d time.Duration) OptionSetter {
	return func(c *Config) {
		if d >= 0 {
			c.MaxTimeWait = d
		}
	}
}

// WithMaxAttempts limits number of retries (-1 for unlimited).
func WithMaxAttempts(n int) OptionSetter {
	return func(c *Config) {
		c.MaxRetries = n
	}
}

// WithExponentialBackoff enables backoff with base/max durations.
func WithExponentialBackoff(base, max time.Duration) OptionSetter {
	return func(c *Config) {
		if base >= 0 && max >= base {
			c.UseBackoff = true
			c.BackoffBase = base
			c.BackoffMax = max
		}
	}
}

// WithErrorFilter sets basic error evaluation callback.
func WithErrorFilter(fn func(error) bool) OptionSetter {
	return func(c *Config) {
		if fn != nil {
			c.ErrorHandler = fn
		}
	}
}

// WithJitter adds delay randomization function.
func WithJitter(fn func(time.Duration) time.Duration) OptionSetter {
	return func(c *Config) {
		if fn != nil {
			c.JitterFunc = fn
		}
	}
}

// WithRetryPolicy sets advanced retry decision function.
func WithRetryPolicy(fn func(int, error) bool) OptionSetter {
	return func(c *Config) {
		if fn != nil {
			c.ShouldRetry = fn
		}
	}
}

// FullJitter applies random delay up to specified duration.
func FullJitter(d time.Duration) time.Duration {
	return time.Duration(rand.Float64() * float64(d))
}

// Exec executes operation with retry logic.
// ctx: Context for cancellation
// op: Operation to execute
// opts: Configuration options
// Returns: nil on success or wrapped error after exhausting retries
func Exec(ctx context.Context, op Operation, opts ...OptionSetter) error {
	cfg := &Config{
		MinTimeWait:  DefaultMinWait,
		MaxTimeWait:  DefaultMaxWait,
		MaxRetries:   DefaultMaxRetries,
		BackoffBase:  DefaultBackoff,
		BackoffMax:   MaxBackoff,
		ErrorHandler: func(err error) bool { return true },
		JitterFunc:   FullJitter,
	}

	for _, opt := range opts {
		opt(cfg)
	}

	if cfg.MinTimeWait > cfg.MaxTimeWait {
		return errors.New("invalid wait time range")
	}
	if cfg.UseBackoff && cfg.BackoffBase > cfg.BackoffMax {
		return errors.New("invalid backoff range")
	}

	rng := rand.New(rand.NewPCG(
		uint64(time.Now().UnixNano()),
		uint64(time.Now().UnixNano()),
	))

	start := time.Now()
	var attempt int
	var lastErr error

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			lastErr = op(ctx, attempt)
			if lastErr == nil {
				return nil
			}

			if !shouldRetry(cfg, attempt, lastErr) {
				return lastErr
			}

			if cfg.MaxRetries == -1 && time.Since(start) > MaxTotalDuration {
				return fmt.Errorf("max duration exceeded: %w", lastErr)
			}

			if cfg.MaxRetries != -1 && attempt >= cfg.MaxRetries {
				return fmt.Errorf("max attempts (%d): %w", cfg.MaxRetries, lastErr)
			}

			sleep := calculateDelay(cfg, attempt, rng)
			timer := time.NewTimer(sleep)

			select {
			case <-ctx.Done():
				timer.Stop()
				return ctx.Err()
			case <-timer.C:
			}

			attempt++
		}
	}
}

// shouldRetry evaluates retry policies in priority order.
func shouldRetry(cfg *Config, attempt int, err error) bool {
	if cfg.ShouldRetry != nil {
		return cfg.ShouldRetry(attempt, err)
	}
	return cfg.ErrorHandler(err)
}

// calculateDelay computes next wait duration with backoff/jitter.
func calculateDelay(cfg *Config, attempt int, rng *rand.Rand) time.Duration {
	var base time.Duration

	if cfg.UseBackoff {
		exp := 1 << uint(attempt)
		base = time.Duration(float64(cfg.BackoffBase) * float64(exp))
		if base > cfg.BackoffMax {
			base = cfg.BackoffMax
		}
	} else {
		base = cfg.MinTimeWait + time.Duration(rng.Float64()*float64(cfg.MaxTimeWait-cfg.MinTimeWait))
	}

	if cfg.JitterFunc != nil {
		base = cfg.JitterFunc(base)
	}

	if base < cfg.MinTimeWait {
		return cfg.MinTimeWait
	}
	if base > cfg.MaxTimeWait {
		return cfg.MaxTimeWait
	}
	return base
}
