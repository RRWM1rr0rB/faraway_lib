package logging

import (
	"log/slog"
	"time"
)

// Constants for log levels (aliases from slog).
const (
	LevelInfo  = slog.LevelInfo
	LevelWarn  = slog.LevelWarn
	LevelError = slog.LevelError
	LevelDebug = slog.LevelDebug
)

// Type aliases for slog types.
type (
	Logger         = slog.Logger
	Attr           = slog.Attr
	Level          = slog.Level
	Handler        = slog.Handler
	Value          = slog.Value
	HandlerOptions = slog.HandlerOptions
	LogValuer      = slog.LogValuer
)

// Handler constructors and global functions (aliases from slog).
var (
	NewTextHandler = slog.NewTextHandler
	NewJSONHandler = slog.NewJSONHandler
	New            = slog.New
	SetDefault     = slog.SetDefault

	StringAttr   = slog.String
	BoolAttr     = slog.Bool
	Float64Attr  = slog.Float64
	AnyAttr      = slog.Any
	DurationAttr = slog.Duration
	IntAttr      = slog.Int
	Int64Attr    = slog.Int64
	Uint64Attr   = slog.Uint64

	GroupValue = slog.GroupValue
	Group      = slog.Group
)

// Float32Attr converts float32 to slog.Float64 attribute.
// WARNING: May lose precision for values outside the ±3.4e38 range.
func Float32Attr(key string, val float32) Attr {
	return slog.Float64(key, float64(val))
}

// UInt32Attr converts uint32 to slog.Int attribute.
// WARNING: May overflow for values > math.MaxInt32.
func UInt32Attr(key string, val uint32) Attr {
	return slog.Int(key, int(val))
}

// Int32Attr converts int32 to slog.Int attribute.
// Safe for all int32 values.
func Int32Attr(key string, val int32) Attr {
	return slog.Int(key, int(val))
}

// TimeAttr formats time.Time to string attribute.
func TimeAttr(key string, time time.Time) Attr {
	return slog.String(key, time.String())
}

// ErrAttr creates an error attribute. Handles nil errors.
func ErrAttr(err error) Attr {
	if err == nil {
		return slog.String("error", "error is nil")
	}
	return slog.String("error", err.Error())
}
