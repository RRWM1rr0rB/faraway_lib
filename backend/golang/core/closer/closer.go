// Package closer provides utilities for managing resource cleanup
// with support for graceful shutdown via OS signals.
package closer

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"os/signal"
	"sync"
)

// Closer represents resources that return an error on closure.
// Matches the standard io.Closer interface.
type Closer = io.Closer

// NoErrCloser represents resources that close without error return.
// Used for resources where close errors can be safely ignored.
type NoErrCloser interface {
	Close()
}

// CloserFunc adapts a function to the Closer interface.
type CloserFunc func() error

// Close implements the Closer interface.
func (f CloserFunc) Close() error {
	return f()
}

// NoErrCloserFunc adapts a function to the NoErrCloser interface.
type NoErrCloserFunc func()

// Close implements the NoErrCloser interface.
func (f NoErrCloserFunc) Close() {
	f()
}

// LIFOCloser manages resources in Last-In-First-Out order.
// Provides thread-safe registration and cleanup of resources.
type LIFOCloser struct {
	mu           sync.Mutex    // Guards access to resources
	closers      []Closer      // Resources with error returns
	noErrClosers []NoErrCloser // Resources without error returns
}

// NewLIFOCloser creates a new LIFOCloser instance.
func NewLIFOCloser() *LIFOCloser {
	return &LIFOCloser{
		closers:      make([]Closer, 0),
		noErrClosers: make([]NoErrCloser, 0),
	}
}

// Add registers error-returning closers for deferred cleanup.
// Thread-safe method.
func (lc *LIFOCloser) Add(closers ...Closer) {
	lc.mu.Lock()
	defer lc.mu.Unlock()
	lc.closers = append(lc.closers, closers...)
}

// AddNoErr registers non-error closers for deferred cleanup.
// Thread-safe method.
func (lc *LIFOCloser) AddNoErr(closers ...NoErrCloser) {
	lc.mu.Lock()
	defer lc.mu.Unlock()
	lc.noErrClosers = append(lc.noErrClosers, closers...)
}

// Close cleans up all registered resources in reverse order (LIFO).
// Returns joined errors if any closers failed.
// Ensures all resources are closed regardless of individual errors.
func (lc *LIFOCloser) Close() error {
	lc.mu.Lock()
	defer lc.mu.Unlock()

	var errs []error

	// Close error-returning resources (reverse order)
	for i := len(lc.closers) - 1; i >= 0; i-- {
		if err := lc.closers[i].Close(); err != nil {
			errs = append(errs, fmt.Errorf("close error: %w", err))
		}
	}

	// Close non-error resources (reverse order)
	for i := len(lc.noErrClosers) - 1; i >= 0; i-- {
		lc.noErrClosers[i].Close()
	}

	if len(errs) > 0 {
		return errors.Join(errs...)
	}
	return nil
}

// CloseOnSignal initiates cleanup when receiving specified OS signals.
// Returns cleanup error or nil. Automatically stops signal catching.
func CloseOnSignal(lc *LIFOCloser, signals ...os.Signal) error {
	done := make(chan os.Signal, 1)
	defer close(done)
	signal.Notify(done, signals...)

	sig := <-done
	log.Printf("Received %v signal, initiating shutdown", sig)
	signal.Stop(done) // Stop catching signals

	return lc.Close()
}

// CloseOnSignalWithContext combines signal handling with context cancellation.
// Initiates cleanup on either received signal or context cancellation.
func CloseOnSignalWithContext(ctx context.Context, lc *LIFOCloser, signals ...os.Signal) error {
	closeCtx, stop := signal.NotifyContext(ctx, signals...)
	defer stop() // Ensure signal catching stops

	<-closeCtx.Done()
	log.Printf("Initiating shutdown: %v", closeCtx.Err())

	return lc.Close()
}

// CloseOnSignalContext creates context-aware shutdown handler.
// Useful for integration with context-based systems like HTTP servers.
func CloseOnSignalContext(lc *LIFOCloser, signals ...os.Signal) func(ctx context.Context) error {
	return func(ctx context.Context) error {
		return CloseOnSignalWithContext(ctx, lc, signals...)
	}
}
