package api

import (
	"log/slog"
	"sync"

	"github.com/sisoputnfrba/tp-golang/utils/config"
	"github.com/sisoputnfrba/tp-golang/utils/log"
)

type Handler struct {
	Log                    *slog.Logger
	Config                 *Config
	EspacioDeUsuario       []byte
	MetricasProcesos       map[int]*MetricasProceso
	mutexInstrucciones     *sync.RWMutex
	Instrucciones          map[int][]Instruccion
	FrameTable             []bool
	TablasProcesos         []*TablasProceso
	ProcesoPorPosicionSwap []int
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
		Config:                 configStruct,
		Log:                    log.BuildLogger(logLevel),
		EspacioDeUsuario:       make([]byte, configStruct.MemorySize),
		MetricasProcesos:       make(map[int]*MetricasProceso),
		Instrucciones:          make(map[int][]Instruccion),
		FrameTable:             make([]bool, configStruct.MemorySize/configStruct.PageSize),
		TablasProcesos:         make([]*TablasProceso, 0),
		ProcesoPorPosicionSwap: make([]int, 0),
		mutexInstrucciones:     &sync.RWMutex{},
	}
}
