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

func StringAttr(key, value string) slog.Attr {
	return slog.String(key, value)
}

func IntAttr(key string, value int) slog.Attr {
	return slog.Int(key, value)
}

func AnyAttr(key string, value any) slog.Attr {
	return slog.Any(key, value)
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
