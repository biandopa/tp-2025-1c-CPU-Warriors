package log

import (
	"log/slog"
	"os"
)

func BuildLogger() *slog.Logger {
	ops := &slog.HandlerOptions{
		AddSource: true,
		Level:     slog.LevelInfo,
	}
	return slog.New(slog.NewJSONHandler(os.Stderr, ops))
}

func ErrAttr(err error) slog.Attr {
	return slog.Any("error", err)
}
