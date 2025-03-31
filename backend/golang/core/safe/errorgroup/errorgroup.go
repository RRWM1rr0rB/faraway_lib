// Package errorgroup provides a panic-safe error group with context awareness
// and error aggregation for concurrent operations.
package errorgroup

import (
	"context"
	"fmt"
	"log/slog"
	"runtime/debug"
	"sync"

	"golang.org/x/sync/errgroup"
)

// SafeGroup enhances errgroup.Group with panic recovery and error aggregation.
type SafeGroup struct {
	eg      *errgroup.Group
	ctx     context.Context
	mu      sync.Mutex
	errs    []error
	recover RecoverFunc
}

// RecoverFunc defines a custom panic recovery handler.
type RecoverFunc func(r any)

// DefaultRecover logs panics with stack traces using slog.
func DefaultRecover(r any) {
	slog.Error("recovered from panic", "panic", r, "stack", string(debug.Stack()))
}

// Option configures a SafeGroup.
type Option func(*SafeGroup)

// WithRecover sets a custom panic recovery handler.
func WithRecover(recover RecoverFunc) Option {
	return func(g *SafeGroup) {
		g.recover = recover
	}
}

// WithContext initializes a SafeGroup with a context and options.
func WithContext(ctx context.Context, opts ...Option) (*SafeGroup, context.Context) {
	eg, ctx := errgroup.WithContext(ctx)
	g := &SafeGroup{
		eg:   eg,
		ctx:  ctx,
		errs: make([]error, 0),
	}
	for _, opt := range opts {
		opt(g)
	}
	if g.recover == nil {
		g.recover = DefaultRecover
	}
	return g, ctx
}

// Go runs a function in a goroutine with panic recovery.
// Errors are collected and can be retrieved via Wait().
func (g *SafeGroup) Go(fn func(ctx context.Context) error) {
	g.eg.Go(func() (err error) {
		defer func() {
			if r := recover(); r != nil {
				g.recover(r)
				err = fmt.Errorf("panic: %v\n%s", r, string(debug.Stack()))
				g.mu.Lock()
				g.errs = append(g.errs, err)
				g.mu.Unlock()
			}
		}()
		err = fn(g.ctx)
		if err != nil {
			g.mu.Lock()
			g.errs = append(g.errs, err)
			g.mu.Unlock()
		}
		return err
	})
}

// Wait blocks until all goroutines complete and returns aggregated errors.
func (g *SafeGroup) Wait() error {
	err := g.eg.Wait()
	if err != nil {
		g.mu.Lock()
		g.errs = append(g.errs, err)
		g.mu.Unlock()
	}
	g.mu.Lock()
	defer g.mu.Unlock()
	if len(g.errs) > 0 {
		return fmt.Errorf("safe group errors: %v", g.errs)
	}
	return nil
}

// Errors returns a copy of all collected errors.
func (g *SafeGroup) Errors() []error {
	g.mu.Lock()
	defer g.mu.Unlock()
	return append([]error{}, g.errs...)
}
