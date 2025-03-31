// Package safe provides panic-safe wrappers for goroutines and functions.
package safe

import (
	"context"
	"fmt"
	"log/slog"
	"runtime/debug"
)

// RecoverFunc handles panics during function execution.
type RecoverFunc func(r any)

// DefaultRecover logs panics with stack traces.
func DefaultRecover(r any) {
	slog.Error("recovered from panic", "panic", r, "stack", string(debug.Stack()))
}

// SafeGo runs a function in a goroutine and recovers panics.
// Errors are sent to the returned channel.
func SafeGo(ctx context.Context, fn func(context.Context) error, recoverFn RecoverFunc) <-chan error {
	errCh := make(chan error, 1)
	if recoverFn == nil {
		recoverFn = DefaultRecover
	}
	go func() {
		defer close(errCh)
		defer func() {
			if r := recover(); r != nil {
				recoverFn(r)
				errCh <- fmt.Errorf("panic: %v", r)
			}
		}()
		if err := fn(ctx); err != nil {
			errCh <- err
		}
	}()
	return errCh
}

// SafeFunc wraps a function with panic recovery.
func SafeFunc(fn func() error, recoverFn RecoverFunc) func() error {
	if recoverFn == nil {
		recoverFn = DefaultRecover
	}
	return func() (err error) {
		defer func() {
			if r := recover(); r != nil {
				recoverFn(r)
				err = fmt.Errorf("panic: %v\n%s", r, string(debug.Stack()))
			}
		}()
		return fn()
	}
}

// SafeCtxFunc wraps a context-aware function with panic recovery.
func SafeCtxFunc(fn func(context.Context) error, recoverFn RecoverFunc) func(context.Context) error {
	if recoverFn == nil {
		recoverFn = DefaultRecover
	}
	return func(ctx context.Context) (err error) {
		defer func() {
			if r := recover(); r != nil {
				recoverFn(r)
				err = fmt.Errorf("panic: %v\n%s", r, string(debug.Stack()))
			}
		}()
		return fn(ctx)
	}
}
