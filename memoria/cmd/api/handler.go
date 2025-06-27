package api

import (
	"log/slog"

	"github.com/sisoputnfrba/tp-golang/utils/config"
	"github.com/sisoputnfrba/tp-golang/utils/log"
)

type Handler struct {
	Log              *slog.Logger
	Config           *Config
	EspacioDeUsuario []byte
	//TablasDePaginas  map[int]map[int]byte // PID -> Pagina -> Frame ???
	MetricasProcesos map[int]*MetricasProceso
	Instrucciones    map[int][]Instruccion
	FrameTable       []bool
	TablasProcesos   []*TablasProceso
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
		Config:           configStruct,
		Log:              log.BuildLogger(logLevel),
		EspacioDeUsuario: make([]byte, configStruct.MemorySize),
		//TablasDePaginas:  make(map[int]map[int]byte),
		MetricasProcesos: make(map[int]*MetricasProceso),
		Instrucciones:    make(map[int][]Instruccion),
		FrameTable:       make([]bool, configStruct.MemorySize/configStruct.PageSize),
		TablasProcesos:   make([]*TablasProceso, 0),
	}
}
