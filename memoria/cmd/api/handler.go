package api

import (
	"log/slog"

	"github.com/sisoputnfrba/tp-golang/utils/config"
	"github.com/sisoputnfrba/tp-golang/utils/log"
)

type Handler struct {
	Log    *slog.Logger
	Config *config.Config
}

func NewHandler(configFile string) *Handler {
	c := config.IniciarConfiguracion(configFile)
	if c == nil {
		panic("Error loading configuration")
	}

	// Initialize the logger with the log level from the configuration
	logLevel := c.LogLevel

	return &Handler{
		Config: c,
		Log:    log.BuildLogger(logLevel),
	}
}
