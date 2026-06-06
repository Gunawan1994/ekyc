package logger

import (
	"context"
	"os"
	"strings"
	"time"

	"github.com/rs/zerolog"
)

// New creates a zerolog.Logger configured by level and format.
//
//   - level: "debug", "info", "warn", "error", "fatal" (default: "info")
//   - format: "pretty" for human-readable console output, anything else for JSON
func New(level, format string) zerolog.Logger {
	lvl, err := zerolog.ParseLevel(strings.ToLower(level))
	if err != nil {
		lvl = zerolog.InfoLevel
	}

	zerolog.SetGlobalLevel(lvl)
	zerolog.TimeFieldFormat = time.RFC3339

	if strings.ToLower(format) == "pretty" {
		writer := zerolog.ConsoleWriter{
			Out:        os.Stdout,
			TimeFormat: time.RFC3339,
		}
		return zerolog.New(writer).With().Timestamp().Logger()
	}

	return zerolog.New(os.Stdout).With().Timestamp().Logger()
}

// WithContext stores logger in ctx and returns the updated context.
func WithContext(ctx context.Context, logger zerolog.Logger) context.Context {
	return logger.WithContext(ctx)
}

// FromContext retrieves the logger stored by WithContext.
// Falls back to a no-op logger when none is found.
func FromContext(ctx context.Context) zerolog.Logger {
	return zerolog.Ctx(ctx).With().Logger()
}
