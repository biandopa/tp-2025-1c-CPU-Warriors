package api

import (
	"encoding/json"
	"net/http"

	"github.com/sisoputnfrba/tp-golang/utils/log"
)

// EjecutarPlanificadores envia un proceso a la Memoria
func (h *Handler) EjecutarPlanificadores(archivoNombre, tamanioProceso string) {
	// Creo un proceso
	//proceso := internal.Proceso{}

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
		h.Planificador.PlanificadorCortoPlazoFIFO()
	case "SJFSD":

	case "SJFD":

	default:
		h.Log.Warn("Algoritmo de corto plazo no reconocido")
	}
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

	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("ok"))

	switch proceso.Instruccion {
	case "INIT_PROC":
		// TODO: Implementar lógica INIT_PROC (Aca creo un nuevo proceso y los paso a new)
	case "IO":
		// TODO: Implementar lógica IO
	case "DUMP_MEMORY":
		// TODO: Implementar lógica DUMP_MEMORY
	case "EXIT":
		// TODO: Implementar lógica EXIT (aca busco el PID en Exec y lo paso a Exit)
	default:
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte("Instrucción no reconocida"))
		return
	}
}
