package planificadores

import (
	"log/slog"
	"net/http"
	"sync"

	"github.com/sisoputnfrba/tp-golang/kernel/internal"
	"github.com/sisoputnfrba/tp-golang/kernel/pkg/cpu"
	"github.com/sisoputnfrba/tp-golang/kernel/pkg/memoria"
)

type Service struct {
	Planificador           *Planificador
	LargoPlazoAlgorithm    string // Algoritmo de largo plazo utilizado
	ShortTermAlgorithm     string // Algoritmo de corto plazo utilizado
	Log                    *slog.Logger
	Memoria                *memoria.Memoria
	CPUsConectadas         []*cpu.Cpu
	CanalEnter             chan struct{}
	canalNuevoProcesoReady chan struct{}
	CanalNuevoProcesoNew   chan *internal.Proceso // Canal para recibir notificaciones de nuevos procesos en NewQueue
	CanalNuevoProcBlocked  chan *internal.Proceso
	CanalNewProcSuspReady  chan *internal.Proceso
	mutexNewQueue          *sync.RWMutex
	mutexReadyQueue        *sync.RWMutex
	mutexCPUsConectadas    *sync.RWMutex
	mutexBlockQueue        *sync.RWMutex
	mutexExecQueue         *sync.RWMutex
	mutexSuspBlockQueue    *sync.RWMutex
	mutexSuspReadyQueue    *sync.RWMutex
	SjfConfig              *SjfConfig
	MedianoPlazoConfig     *MedianoPlazoConfig
	CPUSemaphore           chan struct{} // Semáforo contador para CPUs disponibles
	HttpClient             *http.Client
}

type Planificador struct {
	NewQueue       []*internal.Proceso
	ReadyQueue     []*internal.Proceso
	BlockQueue     []*internal.Proceso
	SuspReadyQueue []*internal.Proceso
	SuspBlockQueue []*internal.Proceso
	ExecQueue      []*internal.Proceso
	ExitQueue      []*internal.Proceso
}

type CpuIdentificacion struct {
	IP     string `json:"ip"`
	Puerto int    `json:"puerto"`
	ID     string `json:"id"`
	Estado bool   `json:"estado"`
}

type SjfConfig struct {
	Alpha           float64 `json:"alpha"`
	InitialEstimate int     `json:"initial_estimate"`
}

type MedianoPlazoConfig struct {
	SuspensionTime int `json:"suspension_time"` // Tiempo de suspensión en milisegundos
}

// NewPlanificador función que sirve para crear una nueva instancia del planificador de procesos. El planificador posee
// varias colas para gestionar los procesos en diferentes estados: New, Ready, Block, Suspended Ready, Suspended Block, Exec y Exit.
func NewPlanificador(log *slog.Logger, ipMemoria, largoPlazoAlgoritmo, cortoPlazoAlgoritmo string,
	puertoMemoria int, sjfConfig *SjfConfig, suspTime int, httpClient *http.Client) *Service {
	return &Service{
		Planificador: &Planificador{
			NewQueue:       make([]*internal.Proceso, 0),
			ReadyQueue:     make([]*internal.Proceso, 0),
			BlockQueue:     make([]*internal.Proceso, 0),
			SuspReadyQueue: make([]*internal.Proceso, 0),
			SuspBlockQueue: make([]*internal.Proceso, 0),
			ExecQueue:      make([]*internal.Proceso, 0),
			ExitQueue:      make([]*internal.Proceso, 0),
		},
		Log:                    log,
		Memoria:                memoria.NewMemoria(ipMemoria, puertoMemoria, log),
		CPUsConectadas:         make([]*cpu.Cpu, 0),
		CanalEnter:             make(chan struct{}),
		SjfConfig:              sjfConfig,
		LargoPlazoAlgorithm:    largoPlazoAlgoritmo,
		ShortTermAlgorithm:     cortoPlazoAlgoritmo,
		canalNuevoProcesoReady: make(chan struct{}, 100),          // Buffer para evitar deadlocks
		CanalNuevoProcesoNew:   make(chan *internal.Proceso, 100), // Buffer para evitar deadlocks
		CanalNewProcSuspReady:  make(chan *internal.Proceso, 100),
		CanalNuevoProcBlocked:  make(chan *internal.Proceso, 100),
		mutexNewQueue:          &sync.RWMutex{},
		mutexReadyQueue:        &sync.RWMutex{},
		mutexBlockQueue:        &sync.RWMutex{},
		mutexExecQueue:         &sync.RWMutex{},
		mutexSuspBlockQueue:    &sync.RWMutex{},
		mutexSuspReadyQueue:    &sync.RWMutex{},
		mutexCPUsConectadas:    &sync.RWMutex{},
		MedianoPlazoConfig: &MedianoPlazoConfig{
			SuspensionTime: suspTime,
		},
		CPUSemaphore: make(chan struct{}, 100), // Inicializamos el semáforo vacío, se llenará cuando se conecten CPUs.
		// Buffer máximo de 100 CPUs
		HttpClient: httpClient,
	}
}
