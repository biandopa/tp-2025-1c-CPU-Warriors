package api

import (
	"log/slog"

	"github.com/sisoputnfrba/tp-golang/cpu/internal"
	"github.com/sisoputnfrba/tp-golang/cpu/pkg/memoria"
	"github.com/sisoputnfrba/tp-golang/utils/config"
	"github.com/sisoputnfrba/tp-golang/utils/log"
)

type Handler struct {
	Log     *slog.Logger
	Config  *Config
	Service *internal.Service
	Memoria *memoria.Memoria
}

func NewHandler(configFile string) *Handler {
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
	logger := log.BuildLogger(logLevel)

	mem := memoria.NewMemoria(configStruct.IpMemory, configStruct.PortMemory, logger)

	return &Handler{
		Config: configStruct,
		Log:    logger,
		Service: internal.NewService(logger, configStruct.IpKernel, configStruct.PortKernel,
			configStruct.TlbEntries, configStruct.CacheEntries,
			configStruct.TlbReplacement, configStruct.CacheReplacement, mem),
		Memoria: mem,
	}
}
