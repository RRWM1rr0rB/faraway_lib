package logging

import (
	"context"
	"log/slog"
	"os"
)

// Default logger configuration constants.
const (
	defaultLevel      = LevelInfo
	defaultAddSource  = true
	defaultIsJSON     = true
	defaultSetDefault = true
)

// NewLogger creates a configurable logger with JSON/Text formatting and source tracking.
func NewLogger(opts ...LoggerOption) *Logger {
	config := &LoggerOptions{
		Level:      defaultLevel,
		AddSource:  defaultAddSource,
		IsJSON:     defaultIsJSON,
		SetDefault: defaultSetDefault,
	}

	for _, opt := range opts {
		opt(config)
	}

	options := &HandlerOptions{
		AddSource: config.AddSource,
		Level:     config.Level,
	}

	var h Handler = NewTextHandler(os.Stdout, options)
	if config.IsJSON {
		h = NewJSONHandler(os.Stdout, options)
	}

	logger := New(h)
	if config.SetDefault {
		SetDefault(logger)
	}

	return logger
}

// LoggerOptions holds configuration for the logger.
type LoggerOptions struct {
	Level      Level
	AddSource  bool
	IsJSON     bool
	SetDefault bool
}

// LoggerOption functional options pattern for logger configuration.
type LoggerOption func(*LoggerOptions)

// WithLevel sets the log level (e.g., "debug") and logs parsing errors.
func WithLevel(level string) LoggerOption {
	return func(o *LoggerOptions) {
		var l Level
		if err := l.UnmarshalText([]byte(level)); err != nil {
			// Log the error using the default logger
			slog.Default().Error(
				"failed to parse log level",
				slog.String("input", level),
				slog.String("default", "info"),
				slog.Any("error", err),
			)
			l = LevelInfo
		}
		o.Level = l
	}
}

// WithAddSource enables/disables source file logging.
func WithAddSource(addSource bool) LoggerOption {
	return func(o *LoggerOptions) {
		o.AddSource = addSource
	}
}

// WithIsJSON sets the output format to JSON.
func WithIsJSON(isJSON bool) LoggerOption {
	return func(o *LoggerOptions) {
		o.IsJSON = isJSON
	}
}

// WithSetDefault sets the logger as the default.
func WithSetDefault(setDefault bool) LoggerOption {
	return func(o *LoggerOptions) {
		o.SetDefault = setDefault
	}
}

// WithAttrs adds attributes to the logger in the context.
func WithAttrs(ctx context.Context, attrs ...Attr) *Logger {
	logger := L(ctx)
	for _, attr := range attrs {
		logger = logger.With(attr)
	}
	return logger
}

// L retrieves the logger from the context.
func L(ctx context.Context) *Logger {
	return loggerFromContext(ctx)
}

// Default returns the global default logger.
func Default() *Logger {
	return slog.Default()
}
