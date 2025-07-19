package api

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/sisoputnfrba/tp-golang/utils/config"
	"github.com/sisoputnfrba/tp-golang/utils/log"
)

type Handler struct {
	Nombre     string
	Log        *slog.Logger
	Config     *Config
	HttpClient *http.Client
}

func NewHandler(configFile, nombre string) *Handler {
	c := config.IniciarConfiguracion(configFile, &Config{})
	if c == nil {
		panic("Error loading configuration")
	}

	// Cast the configuration to the specific type
	configStruct, ok := c.(*Config)
	if !ok {
		panic("Error casting configuration")
	}

	// Initialize the logger with the log level from the configuration
	logLevel := configStruct.LogLevel

	httpClient := &http.Client{
		Timeout: 2 * time.Minute,
	}

	return &Handler{
		Nombre:     nombre,
		Config:     configStruct,
		Log:        log.BuildLogger(logLevel),
		HttpClient: httpClient,
	}
}
