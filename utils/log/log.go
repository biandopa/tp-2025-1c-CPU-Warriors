package log

import (
	"log/slog"
	"os"
	"strings"
)

func BuildLogger(level string) *slog.Logger {
	ops := &slog.HandlerOptions{
		AddSource: true,
		Level:     getLevelByName(level),
	}
	return slog.New(slog.NewJSONHandler(os.Stderr, ops))
}

func ErrAttr(err error) slog.Attr {
	return slog.Any("error", err)
}

func getLevelByName(level string) slog.Level {
	switch strings.ToLower(level) {
	case "debug":
		return slog.LevelDebug
	case "info":
		return slog.LevelInfo
	case "warn":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}
