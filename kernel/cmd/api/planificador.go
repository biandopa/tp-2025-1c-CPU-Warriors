package api

import (
	"encoding/json"
	"net/http"
	"sync"

	"github.com/sisoputnfrba/tp-golang/kernel/internal"
	"github.com/sisoputnfrba/tp-golang/utils/log"
)

type rtaCPU struct {
	PID         int      `json:"pid"`
	PC          int      `json:"pc"`
	Instruccion string   `json:"instruccion"`
	Args        []string `json:"args,omitempty"`
}

// EjecutarPlanificadores envia un proceso a la Memoria
func (h *Handler) EjecutarPlanificadores(archivoNombre, tamanioProceso string) {
	// Creo un proceso
	proceso := internal.Proceso{
		PCB: &internal.PCB{
			PID:            0,
			PC:             0,
			MetricasTiempo: map[internal.Estado]*internal.EstadoTiempo{},
			MetricasEstado: map[internal.Estado]int{},
		},
	}

	// TODO: Hacer un switch para elegir un planificador y que ejecute interfaces
	// TODO: Hacer que los planificadores se ejecuten en async

	switch h.Config.ReadyIngressAlgorithm {
	case "FIFO":
		go h.Planificador.PlanificadorLargoPlazoFIFO(archivoNombre, tamanioProceso)
	case "PMCP":

	default:
		h.Log.Warn("Algoritmo de largo plazo no reconocido")
	}

	switch h.Config.SchedulerAlgorithm {
	case "FIFO":
		go h.Planificador.PlanificadorCortoPlazoFIFO()
	case "SJFSD":

	case "SJFD":

	default:
		h.Log.Warn("Algoritmo de corto plazo no reconocido")
	}

	if len(h.Planificador.Planificador.NewQueue) == 0 {
		// Si la cola de New está vacía, la inicializo
		h.Planificador.Planificador.NewQueue = make([]*internal.Proceso, 1)
	}
	h.Planificador.Planificador.NewQueue[0] = &proceso
}

// ejecutarPlanificadorCortoPlazo selecciona el planificador de corto plazo a utilizar y lo ejecuta como una goroutine.
func (h *Handler) ejecutarPlanificadorCortoPlazo() {
	switch h.Config.ReadyIngressAlgorithm {
	case "FIFO":
		go h.Planificador.PlanificadorCortoPlazoFIFO()
	case "SJFSD":

	case "SJFD":
	case "PMCP":

	default:
		h.Log.Warn("Algoritmo no reconocido")
	}
}

func (h *Handler) RespuestaProcesoCPU(w http.ResponseWriter, r *http.Request) {
	decoder := json.NewDecoder(r.Body)
	var syscall rtaCPU

	err := decoder.Decode(&syscall)
	if err != nil {
		h.Log.Error("Error al decodificar la RTA del Proceso",
			log.ErrAttr(err),
		)
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte("Error al decodificar la RTA del Proceso"))
	}

	h.Log.Debug("Me llego la RTA del Proceso",
		log.AnyAttr("syscall", syscall),
	)

	switch syscall.Instruccion {
	case "INIT_PROC":
		mu := sync.Mutex{}
		if len(syscall.Args) < 2 {
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte("Error: no se recibieron los argumentos necesarios (archivo y tamaño)"))
			return
		}

		// Creo un proceso hijo
		proceso := internal.Proceso{
			PCB: &internal.PCB{
				PID:            1,
				PC:             0,
				MetricasTiempo: map[internal.Estado]*internal.EstadoTiempo{},
				MetricasEstado: map[internal.Estado]int{},
			},
		}

		// TODO: Agregar channel NewQueue
		mu.Lock()
		h.Planificador.Planificador.NewQueue = append(h.Planificador.Planificador.NewQueue, &proceso)
		mu.Unlock()
	case "IO":
		// TODO: Implementar lógica IO
		/* Primero verifica que existe el IO. Si no existe, se manda a EXIT.
		Si existe y está ocupado, se manda a Blocked. Veremos...*/
	case "DUMP_MEMORY":
		// TODO: Implementar lógica DUMP_MEMORY
		// Nota: Este todavía no!!!!!
		/* Esta bloquea el proceso. En caso de error se envía a exit y sino se pasa a Ready*/
	case "EXIT":
		go h.Planificador.FinalizarProceso(syscall.PID)
	default:
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte("Instrucción no reconocida"))
		return
	}

	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("ok"))
}
