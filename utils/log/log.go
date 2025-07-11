package log

import (
	"fmt"
	"io"
	"log/slog"
	"os"
	"strings"
	"time"
)

func BuildLogger(level string) *slog.Logger {
	ops := &slog.HandlerOptions{
		//AddSource: true,
		Level: getLevelByName(level),
	}

	output := configurarLoggerOutput()
	logger := slog.New(slog.NewJSONHandler(output, ops))

	return logger
}

func configurarLoggerOutput() io.Writer {
	workingDir, err := os.Getwd()
	if err != nil {
		panic(fmt.Sprintf("Error al obtener el directorio de trabajo: %v", err))
	}

	now := time.Now()
	nowStr := now.Format("2006-01-02T15-04-05")

	logFile, err := os.OpenFile(workingDir+"/tp-"+nowStr+".log", os.O_CREATE|os.O_APPEND|os.O_RDWR, 0666)
	if err != nil {
		panic(err)
	}

	return io.MultiWriter(os.Stdout, logFile)
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
