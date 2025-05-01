package api

import (
	"encoding/json"
	"net/http"

	"github.com/sisoputnfrba/tp-golang/kernel/internal"
	"github.com/sisoputnfrba/tp-golang/utils/log"
)

// EnviarProceso envia un proceso a la Memoria
func (h *Handler) EnviarProceso(archivoNombre, tamanioProceso, args string) {
	// Creo un proceso
	//proceso := internal.Proceso{}

	// TODO: Hacer un switch para elegir un planificador y que ejecute interfaces
	if h.Config.SchedulerAlgorithm == "FIFO" {
		h.Planificador.PlanificadorLargoPlazoFIFO(args)

		// Se ejecuta algun otro planificador

		//h.Planificador.FinalizarProceso(proceso)
	}

}

func (h *Handler) RespuestaProcesoCPU(w http.ResponseWriter, r *http.Request) {
	decoder := json.NewDecoder(r.Body)
	var proceso internal.Proceso

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
}
