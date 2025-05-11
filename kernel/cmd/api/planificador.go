package api

import (
	"encoding/json"
	"net/http"

	"github.com/sisoputnfrba/tp-golang/kernel/internal"
	"github.com/sisoputnfrba/tp-golang/utils/log"
)

// EjecutarPlanificadores envia un proceso a la Memoria
func (h *Handler) EjecutarPlanificadores(archivoNombre, tamanioProceso string) {
	// Creo un proceso
	proceso := internal.Proceso{
		PCB: &internal.PCB{
			PID:            0,
			ProgramCounter: 0,
			MetricasTiempo: map[internal.Estado]*internal.EstadoTiempo{},
			MetricasEstado: map[internal.Estado]int{},
		},
	}

	// TODO: Hacer un switch para elegir un planificador y que ejecute interfaces
	// TODO: Hacer que los planificadores se ejecuten en async

	switch h.Config.ReadyIngressAlgorithm {
	case "FIFO":
		go h.Planificador.PlanificadorLargoPlazoFIFO()
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

//LA IDEA ES QUE LA CONSUMA EL PLANIFICADOR CORTO

// SeleccionarPlanificador selecciona el planificador de corto plazo a utilizar.
func (h *Handler) SeleccionarPlanificador() {

	switch h.Config.ReadyIngressAlgorithm {
	case "FIFO":
		h.Planificador.PlanificadorCortoPlazoFIFO()
	case "SJFSD":

	case "SJFD":
	case "PMCP":

	default:
		h.Log.Warn("Algoritmo no reconocido")
	}

}

/*func (h *Handler) PlanificadorCortoPlazoFIFO() {

	h.Log.Debug("Entre Al PLannificador")

	//TODO: MANDARLO AL PLANFICADOR Y RECIBIR EL PORCESO Y LA CPU DONDE DEBE EJECUTAR
	//PlanificadorCortoPlazoFIFO
	cpu := CPUIdentificacion{
		IP:     "127.0.0.1",
		Puerto: 8004,
		ID:     "CPU-1",
		ESTADO: true,
	}
	//TODO: RECIBIR EL PROCESO A ENVIAR A CPU
	h.enviarProcesoACPU(cpu)
}*/

//ESto devuelve el PID + PC + alguno de estos
//IO 25000
//INIT_PROC proceso1 256
//DUMP_MEMORY
//EXIT

type rtaCPU struct {
	PID         int      `json:"pid"`
	PC          int      `json:"pc"`
	Instruccion string   `json:"instruccion"`
	Args        []string `json:"args,omitempty"`
}

func (h *Handler) RespuestaProcesoCPU(w http.ResponseWriter, r *http.Request) {
	decoder := json.NewDecoder(r.Body)
	var proceso rtaCPU

	err := decoder.Decode(&proceso)
	if err != nil {
		h.Log.Error("Error al decodificar la RTA del Proceso",
			log.ErrAttr(err),
		)
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte("Error al decodificar la RTA del Proceso"))
	}

	h.Log.Debug("Me llego la RTA del Proceso",
		log.AnyAttr("proceso", proceso),
	)

	// TODO: Implementar lógica para manejar la respuesta del proceso
	switch proceso.Instruccion {
	case "INIT_PROC":
		// TODO: Implementar lógica INIT_PROC (Aca creo un nuevo proceso y los paso a new)
		/* Crea otro nuevo proceso y lo agrega a la cola de New.
		Vuelve a ejecutar en el planificador de Largo Plazo creando su PCB en este nuevo proceso
		y el padre sigue en su estado*/
	case "IO":
		// TODO: Implementar lógica IO
		/* Primero verifica que existe el IO. Si no existe, se manda a EXIT.
		Si existe y está ocupado, se manda a Blocked. Veremos...*/
	case "DUMP_MEMORY":
		// TODO: Implementar lógica DUMP_MEMORY
		// Nota: Este todavía no!!!!!
		/* Esta bloquea el proceso. En caso de error se envía a exit y sino se pasa a Ready*/
	case "EXIT":
		// TODO: Implementar lógica EXIT (aca busco el PID en Exec y lo paso a Exit)
		/* Como le devuelve el PID, tiene que buscar al proceso en la cola de exec y terminarlo.
		Hay que cambiar el estado de la CPU tmb!!!!!*/
		go h.Planificador.FinalizarProceso(proceso.PID)
	default:
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte("Instrucción no reconocida"))
		return
	}

	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("ok"))

}
