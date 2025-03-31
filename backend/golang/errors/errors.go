// Package errors provides utilities for creating, combining, and inspecting errors.
// It builds on the standard errors package and adds multi-error support via go-multierror.
package errors

import (
	"errors"
	"fmt"

	"github.com/hashicorp/go-multierror"
)

// New creates a new error with the given message.
// Returns nil if msg is empty, following errors.New behavior.
func New(msg string) error {
	if msg == "" {
		return nil
	}
	return errors.New(msg)
}

// Errorf creates a formatted error with the given message and arguments.
// Use %w in the format string to wrap another error.
func Errorf(format string, args ...any) error {
	return fmt.Errorf(format, args...)
}

// Wrap wraps an error with a message, preserving the original as a cause.
// If err is nil, returns nil.
func Wrap(err error, msg string) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("%s: %w", msg, err)
}

// Append combines multiple errors into a single multi-error.
// Returns nil if all errors are nil.
func Append(err error, errs ...error) error {
	return multierror.Append(err, errs...)
}

// Flatten simplifies a multi-error into a single error.
// If err is not a multi-error, returns it unchanged.
func Flatten(err error) error {
	return multierror.Flatten(err)
}

// Prefix adds a prefix to an error's message(s).
// If err is a multi-error, prefixes all underlying errors.
func Prefix(err error, prefix string) error {
	return multierror.Prefix(err, prefix)
}

// Join combines multiple errors into a single error using errors.Join.
// Returns nil if all errors are nil.
func Join(errs ...error) error {
	return errors.Join(errs...)
}

// Unwrap returns the underlying error if err supports unwrapping.
// Returns nil if no underlying error exists.
func Unwrap(err error) error {
	return errors.Unwrap(err)
}

// Is reports whether err or any error in its chain matches target.
func Is(err, target error) bool {
	return errors.Is(err, target)
}

// As attempts to cast err to the type of target, returning true if successful.
// The target must be a pointer to an error type (e.g., *MyError).
func As(err error, target any) bool {
	return errors.As(err, target)
}

// Combine combines an error with a message, returning a new error.
// If err is nil, returns a new error with the given message.
// Otherwise, returns a new multi-error with err as one of the errors.
func Combine(err error, msg string) error {
	if err == nil {
		return New(msg)
	}
	return Append(err, New(msg))
}
