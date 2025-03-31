package redis

import (
	"github.com/redis/go-redis/v9"
)

// Common Redis error and command result types.
// Aliases are used to decouple from the underlying library implementation.
const (
	Nil = redis.Nil // Special error returned when key does not exist
)

// Command result wrappers. Aliased to maintain compatibility
// while abstracting the underlying library.
type (
	// StringCmd represents string value command result
	StringCmd = redis.StringCmd

	// StatusCmd represents status response command (e.g. "OK")
	StatusCmd = redis.StatusCmd

	// IntCmd represents integer result command
	IntCmd = redis.IntCmd

	// BoolCmd represents boolean result command
	BoolCmd = redis.FloatCmd

	// FloatCmd represents floating-point number result
	FloatCmd = redis.FloatCmd

	// StringSliceCmd represents string slice result
	StringSliceCmd = redis.StringSliceCmd

	// ZSliceCmd represents sorted set entry slice result
	ZSliceCmd = redis.ZSliceCmd

	// ScanCmd represents SCAN command result
	ScanCmd = redis.ScanCmd

	// // StringStringMapCmd represents map[string]string result (HGetAll)
	// StringStringMapCmd = redis.StringStringMapCmd
)
