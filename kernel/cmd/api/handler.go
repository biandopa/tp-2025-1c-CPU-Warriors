package api

import (
	"log/slog"

	"github.com/sisoputnfrba/tp-golang/kernel/internal/planificadores"
	"github.com/sisoputnfrba/tp-golang/utils/config"
	"github.com/sisoputnfrba/tp-golang/utils/log"
)

type Handler struct {
	Log           *slog.Logger
	Config        *Config
	CPUConectadas []CPUIdentificacion
	Planificador  *planificadores.Service
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

	return &Handler{
		Config:        configStruct,
		Log:           log.BuildLogger(logLevel),
		CPUConectadas: []CPUIdentificacion{},
		Planificador:  planificadores.NewPlanificador,
	}
}
