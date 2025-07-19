package api

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/sisoputnfrba/tp-golang/kernel/internal/planificadores"
	"github.com/sisoputnfrba/tp-golang/utils/config"
	"github.com/sisoputnfrba/tp-golang/utils/log"
	uniqueid "github.com/sisoputnfrba/tp-golang/utils/unique-id"
)

type Handler struct {
	Log          *slog.Logger
	Config       *Config
	Planificador *planificadores.Service
	UniqueID     *uniqueid.UniqueID
	HttpClient   *http.Client
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

	httpClient := &http.Client{
		Timeout: 2 * time.Minute,
	}

	return &Handler{
		Config: configStruct,
		Log:    logger,
		Planificador: planificadores.NewPlanificador(
			logger, configStruct.IpMemory,
			configStruct.ReadyIngressAlgorithm, configStruct.SchedulerAlgorithm,
			configStruct.PortMemory,
			&planificadores.SjfConfig{
				Alpha:           configStruct.Alpha,
				InitialEstimate: configStruct.InitialEstimate,
			},
			configStruct.SuspensionTime,
			httpClient,
		),
		UniqueID:   uniqueid.Init(),
		HttpClient: httpClient,
	}
}
