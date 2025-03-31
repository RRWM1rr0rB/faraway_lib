// Package pointer provides type-safe utilities for working with pointers,
// including creation, dereferencing, and value manipulation.
package pointer

import (
	"reflect"

	"trade/app/pkg/errors"
)

// ToPointer returns a pointer to a copy of the given value.
// Prevents unintended aliasing by creating new memory allocation.
func ToPointer[T any](v T) *T {
	return &v
}

// FromPointer safely dereferences a pointer.
// Returns:
//   - value: Dereferenced value or zero value
//   - ok: True if pointer was non-nil
func FromPointer[T any](v *T) (value T, ok bool) {
	if v == nil {
		return *new(T), false
	}
	return *v, true
}

// FromPointerOr provides fallback value for nil pointers.
// Useful for default value handling.
func FromPointerOr[T any](v *T, fallback T) T {
	if v == nil {
		return fallback
	}
	return *v
}

// IsNil safely checks if pointer is nil.
// Works with interface pointers and generic types.
func IsNil[T any](v *T) bool {
	return v == nil
}

// Swap exchanges values between two non-nil pointers.
// Panics if either pointer is nil to prevent undefined behavior.
func Swap[T any](a, b *T) {
	if a == nil || b == nil {
		panic("pointer: nil pointer in swap operation")
	}
	*a, *b = *b, *a
}

// Copy creates a new pointer with duplicated value.
// Limitations:
//   - Returns nil for nil input
//   - Only works with comparable types
//   - Shallow copy for complex structures
func Copy[T any](v *T) (*T, error) {
	if v == nil {
		return nil, nil
	}

	// Reflection-based type validation
	rv := reflect.ValueOf(*v)
	if !rv.Type().Comparable() {
		return nil, errors.New("pointer: unsupported non-comparable type")
	}

	cpy := *v // Performs shallow copy
	return &cpy, nil
}

// Set updates pointer value if non-nil.
// Returns success status for chaining operations.
func Set[T any](p *T, v T) (ok bool) {
	if p != nil {
		*p = v
		return true
	}
	return false
}
