package planificadores

import (
	"log/slog"

	"github.com/sisoputnfrba/tp-golang/kernel/internal"
	"github.com/sisoputnfrba/tp-golang/kernel/pkg/memoria"
)

type Service struct {
	Planificador  *Planificador
	Log           *slog.Logger
	Memoria       *memoria.Memoria
	CPUConectadas []*CpuIdentificacion // TODO: Ver si hace falta exponerlo o se puede hacer privado
	CanalEnter    chan struct{}
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
	ESTADO bool   `json:"estado"`
}

// NewPlanificador función que sirve para crear una nueva instancia del planificador de procesos. El planificador posee
// varias colas para gestionar los procesos en diferentes estados: New, Ready, Block, Suspended Ready, Suspended Block, Exec y Exit.
func NewPlanificador(log *slog.Logger, ipMemoria string, puertoMemoria int, cpus []*CpuIdentificacion) *Service {
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
		Log:           log,
		Memoria:       memoria.NewMemoria(ipMemoria, puertoMemoria, log),
		CPUConectadas: cpus,
		CanalEnter:    make(chan struct{}),
	}
}
