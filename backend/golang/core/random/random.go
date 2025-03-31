// Package random provides concurrency-safe utilities for generating various
// random values with configurable sources. Uses math/rand/v2 for performance.
package random

import (
	"errors"
	"fmt"
	"math/rand/v2"
	"time"
)

// defaultRand is the shared random source seeded with current time.
// Thread-safe according to math/rand/v2 specifications.
var defaultRand = rand.New(rand.NewPCG(
	uint64(time.Now().UnixNano()),
	uint64(time.Now().UnixNano()),
))

// RandInt generates random integer in [min, max) range.
// Args:
//   - r: Optional random source (uses default if nil)
//   - min: Inclusive lower bound
//   - max: Exclusive upper bound
//
// Returns:
//   - int: Random value
//   - error: If max <= min
func RandInt(r *rand.Rand, min, max int) (int, error) {
	if max <= min {
		return 0, errors.New("random: max must be greater than min")
	}
	if r == nil {
		r = defaultRand
	}
	return min + r.IntN(max-min), nil
}

// RandInt64 generates random int64 in [min, max) range.
// Args:
//   - r: Optional random source
//   - min: Inclusive lower bound
//   - max: Exclusive upper bound
//
// Returns:
//   - int64: Random value
//   - error: If max <= min
func RandInt64(r *rand.Rand, min, max int64) (int64, error) {
	if max <= min {
		return 0, errors.New("random: max must be greater than min")
	}
	if r == nil {
		r = defaultRand
	}
	return min + r.Int64N(max-min), nil
}

// RandFloat64 generates random float64 in [min, max) range.
// Args:
//   - r: Optional random source
//   - min: Inclusive lower bound
//   - max: Exclusive upper bound
//
// Returns:
//   - float64: Random value
//   - error: If max <= min
func RandFloat64(r *rand.Rand, min, max float64) (float64, error) {
	if max <= min {
		return 0, errors.New("random: max must be greater than min")
	}
	if r == nil {
		r = defaultRand
	}
	return min + r.Float64()*(max-min), nil
}

// RandIP generates valid random IPv4 address string.
// Args:
//   - r: Optional random source
//
// Returns:
//   - string: IP in "a.b.c.d" format
//
// Notes:
//   - All octets in 0-255 range
func RandIP(r *rand.Rand) (string, error) {
	if r == nil {
		r = defaultRand
	}

	randOctet := func() int { return r.IntN(256) }
	return fmt.Sprintf("%d.%d.%d.%d",
		randOctet(), randOctet(), randOctet(), randOctet(),
	), nil
}

// RandomDate generates time between min and max (inclusive).
// Args:
//   - r: Optional random source
//   - min: Start time (inclusive)
//   - max: End time (inclusive)
//
// Returns:
//   - time.Time: Random date
//   - error: If max < min
func RandomDate(r *rand.Rand, min, max time.Time) (time.Time, error) {
	if max.Before(min) {
		return time.Time{}, errors.New("random: max must be after min")
	}
	if r == nil {
		r = defaultRand
	}

	delta := max.Unix() - min.Unix()
	sec, _ := RandInt64(r, 0, delta+1)
	return min.Add(time.Duration(sec) * time.Second), nil
}

// DefaultSet contains alphanumeric characters for RandString
var DefaultSet = []byte("ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789")

// RandString generates random string from character set.
// Args:
//   - r: Optional random source
//   - n: String length
//   - set: Allowed characters (uses DefaultSet if empty)
//
// Returns:
//   - string: Generated string
//   - error: If n < 0
func RandString(r *rand.Rand, n int, set []byte) (string, error) {
	if n < 0 {
		return "", errors.New("random: n must be non-negative")
	}
	if len(set) == 0 {
		set = DefaultSet
	}
	if r == nil {
		r = defaultRand
	}

	b := make([]byte, n)
	for i := range b {
		b[i] = set[r.IntN(len(set))]
	}
	return string(b), nil
}

// RandomCase selects random element from arguments.
// Args:
//   - r: Optional random source
//   - args: Values to choose from
//
// Returns:
//   - interface{}: Selected value
//   - error: If no arguments provided
func RandomCase(r *rand.Rand, args ...interface{}) (interface{}, error) {
	if len(args) == 0 {
		return nil, errors.New("random: no arguments provided")
	}
	if r == nil {
		r = defaultRand
	}

	return args[r.IntN(len(args))], nil
}

// RandomBool generates random boolean value.
// Args:
//   - r: Optional random source
//
// Returns:
//   - bool: Random true/false
func RandomBool(r *rand.Rand) bool {
	if r == nil {
		r = defaultRand
	}
	return r.IntN(2) == 0
}
