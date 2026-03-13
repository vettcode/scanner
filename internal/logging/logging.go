package logging

import (
	"io"
	"log/slog"
	"os"
)

// Init sets up the global slog logger with the given level.
// Supported levels: "debug", "info", "warn", "error".
func Init(level string, w io.Writer) *slog.Logger {
	if w == nil {
		w = os.Stderr
	}

	var lvl slog.Level
	switch level {
	case "debug":
		lvl = slog.LevelDebug
	case "warn":
		lvl = slog.LevelWarn
	case "error":
		lvl = slog.LevelError
	default:
		lvl = slog.LevelInfo
	}

	handler := slog.NewTextHandler(w, &slog.HandlerOptions{
		Level: lvl,
	})
	logger := slog.New(handler)
	slog.SetDefault(logger)
	return logger
}

// ParseLevel converts a level string to slog.Level.
func ParseLevel(level string) slog.Level {
	switch level {
	case "debug":
		return slog.LevelDebug
	case "warn":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}
