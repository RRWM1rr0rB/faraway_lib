// Package array provides utilities for manipulating and inspecting slices,
// with a focus on data processing and transformation.
package array

import (
	"errors"
	"fmt"
	"math/rand/v2"
	"sort"
	"strings"

	"golang.org/x/exp/constraints"
)

// Contains checks if an element is present in a slice.
// Works with any comparable type (int, string, etc.).
func Contains[T comparable](s []T, e T) bool {
	for _, a := range s {
		if a == e {
			return true
		}
	}
	return false
}

// RemoveByValue removes the first occurrence of a value from a slice.
// Returns the new slice or the original if value not found.
// Does not modify the original slice.
func RemoveByValue[T comparable](s []T, value T) []T {
	for i, v := range s {
		if v == value {
			return append(s[:i], s[i+1:]...)
		}
	}
	return s
}

// RemoveByIndex removes an element at the specified index from a slice.
// Returns the new slice and an error if index is out of bounds.
// Does not modify the original slice.
func RemoveByIndex[T any](s []T, index int) ([]T, error) {
	if index < 0 || index >= len(s) {
		return nil, errors.New("array: index out of bounds")
	}
	return append(s[:index], s[index+1:]...), nil
}

// IndexOf returns the first index of a value in a slice, or -1 if not found.
// Works with any comparable type.
func IndexOf[T comparable](s []T, value T) int {
	for i, v := range s {
		if v == value {
			return i
		}
	}
	return -1
}

// AreIdentical checks if two slices contain the same elements with identical counts.
// Works with any comparable type.
func AreIdentical[T comparable](x, y []T) bool {
	if len(x) != len(y) {
		return false
	}
	counts := make(map[T]int, len(x))
	for _, v := range x {
		counts[v]++
	}
	for _, v := range y {
		if _, ok := counts[v]; !ok {
			return false
		}
		counts[v]--
		if counts[v] == 0 {
			delete(counts, v)
		}
	}
	return len(counts) == 0
}

// Filter returns a new slice with elements that satisfy the predicate.
func Filter[T any](s []T, keep func(T) bool) []T {
	result := make([]T, 0, len(s))
	for _, v := range s {
		if keep(v) {
			result = append(result, v)
		}
	}
	return result
}

// Map applies a transformation function to each element, returning a new slice.
func Map[T, U any](s []T, transform func(T) U) []U {
	result := make([]U, len(s))
	for i, v := range s {
		result[i] = transform(v)
	}
	return result
}

// Uniq removes duplicates from a slice, preserving order.
// Works with any comparable type.
func Uniq[T comparable](s []T) []T {
	seen := make(map[T]struct{}, len(s))
	result := make([]T, 0, len(s))
	for _, v := range s {
		if _, ok := seen[v]; !ok {
			seen[v] = struct{}{}
			result = append(result, v)
		}
	}
	return result
}

// Reverse returns a new slice with elements in reversed order.
func Reverse[T any](s []T) []T {
	result := make([]T, len(s))
	for i, v := range s {
		result[len(s)-1-i] = v
	}
	return result
}

// Join concatenates slice elements into a string with a separator.
// Works with any type that can be formatted with %v.
func Join[T any](s []T, sep string) string {
	if len(s) == 0 {
		return ""
	}
	strs := Map(s, func(v T) string { return fmt.Sprintf("%v", v) })
	return strings.Join(strs, sep)
}

// JoinString concatenates elements using their String() method.
// Requires T to implement fmt.Stringer.
func JoinString[T fmt.Stringer](s []T, sep string) string {
	strs := make([]string, len(s))
	for i, v := range s {
		strs[i] = v.String()
	}
	return strings.Join(strs, sep)
}

// Sort sorts the slice using the provided less function.
// For ordered types, use: Sort(s, func(a, b T) bool { return a < b })
func Sort[T any](s []T, less func(a, b T) bool) []T {
	result := make([]T, len(s))
	copy(result, s)
	sort.Slice(result, func(i, j int) bool {
		return less(result[i], result[j])
	})
	return result
}

// Concat combines multiple slices into a single slice.
// Works with any type.
func Concat[T any](slices ...[]T) []T {
	totalLen := 0
	for _, s := range slices {
		totalLen += len(s)
	}
	result := make([]T, 0, totalLen)
	for _, s := range slices {
		result = append(result, s...)
	}
	return result
}

// Find returns the first element in a slice that satisfies the predicate.
// Returns the element and a boolean indicating if it was found.
func Find[T any](s []T, match func(T) bool) (T, bool) {
	var zero T
	for _, v := range s {
		if match(v) {
			return v, true
		}
	}
	return zero, false
}

// Split divides a slice into chunks of the specified size.
// The last chunk may be smaller if the length is not evenly divisible.
func Split[T any](s []T, size int) ([][]T, error) {
	if size <= 0 {
		return nil, errors.New("array: split size must be positive")
	}
	if len(s) == 0 {
		return [][]T{}, nil
	}
	result := make([][]T, 0, (len(s)+size-1)/size)
	for i := 0; i < len(s); i += size {
		end := i + size
		if end > len(s) {
			end = len(s)
		}
		result = append(result, s[i:end])
	}
	return result, nil
}

// MinMax finds the minimum and maximum values in a slice.
// Works with ordered types (int, float64, string, etc.).
func MinMax[T constraints.Ordered](s []T) (min T, max T, err error) {
	if len(s) == 0 {
		return min, max, errors.New("array: cannot find min/max of empty slice")
	}
	min, max = s[0], s[0]
	for _, v := range s[1:] {
		if v < min {
			min = v
		}
		if v > max {
			max = v
		}
	}
	return min, max, nil
}

// Every checks if all elements in a slice satisfy the predicate.
func Every[T any](s []T, test func(T) bool) bool {
	for _, v := range s {
		if !test(v) {
			return false
		}
	}
	return true
}

// Some checks if at least one element in a slice satisfies the predicate.
func Some[T any](s []T, test func(T) bool) bool {
	for _, v := range s {
		if test(v) {
			return true
		}
	}
	return false
}

// GroupBy groups elements by a key function, returning a map of key to slice.
// The key type K must be comparable; T can be any type.
func GroupBy[T any, K comparable](s []T, keyFunc func(T) K) map[K][]T {
	result := make(map[K][]T)
	for _, v := range s {
		key := keyFunc(v)
		result[key] = append(result[key], v)
	}
	return result
}

// Reduce applies a reduction function over the slice, accumulating a result.
// Starts with an initial value; T and U can be any types.
func Reduce[T, U any](s []T, reducer func(U, T) U, initial U) U {
	result := initial
	for _, v := range s {
		result = reducer(result, v)
	}
	return result
}

// Shuffle randomly reorders a slice.
// Uses math/rand/v2 for randomization; does not modify the original slice.
func Shuffle[T any](s []T) []T {
	result := make([]T, len(s))
	copy(result, s)
	rand.Shuffle(len(result), func(i, j int) {
		result[i], result[j] = result[j], result[i]
	})
	return result
}

// Partition splits a slice into two based on a predicate.
// Returns elements that pass and fail the test, respectively.
func Partition[T any](s []T, test func(T) bool) (pass, fail []T) {
	pass = make([]T, 0, len(s)/2)
	fail = make([]T, 0, len(s)/2)
	for _, v := range s {
		if test(v) {
			pass = append(pass, v)
		} else {
			fail = append(fail, v)
		}
	}
	return pass, fail
}

// Pair represents a pair of values of two different types.
type Pair[T, U any] struct {
	First  T
	Second U
}

// Zip combines two slices into a slice of Pair structs.
// Stops at the length of the shorter slice.
func Zip[T, U any](s1 []T, s2 []U) []Pair[T, U] {
	minLen := min(len(s1), len(s2))
	result := make([]Pair[T, U], minLen)
	for i := 0; i < minLen; i++ {
		result[i] = Pair[T, U]{First: s1[i], Second: s2[i]}
	}
	return result
}

// DistinctCount returns a map of unique elements to their frequencies.
// Works with any comparable type.
func DistinctCount[T comparable](s []T) map[T]int {
	counts := make(map[T]int, len(s))
	for _, v := range s {
		counts[v]++
	}
	return counts
}

// Take returns the first n elements of a slice.
// Returns all if n >= len(s), or an empty slice with error if n < 0.
func Take[T any](s []T, n int) ([]T, error) {
	if n < 0 {
		return []T{}, errors.New("array: take count must be non-negative")
	}
	if n >= len(s) {
		return append([]T{}, s...), nil
	}
	return append([]T{}, s[:n]...), nil
}
