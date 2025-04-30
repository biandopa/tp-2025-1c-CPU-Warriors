package api

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/sisoputnfrba/tp-golang/kernel/internal"
)

// EnviarProceso envia un proceso a la Memoria
func (h *Handler) EnviarProceso(archivoNombre, tamanioProceso, args string) {
	// Creo un proceso
	proceso := internal.Proceso{}

	if h.Config.SchedulerAlgorithm == "FIFO" {
		planificador := internal.Planificador{
			NewQueue: []*internal.Proceso{
				&proceso,
			},
		}

		planificador.PlanificadorLargoPlazoFIFO(args)

		// Se ejecuta algun otro planificador

		planificador.FinalizarProceso(proceso)
	}

}

func (h *Handler) RespuestaProcesoCPU(w http.ResponseWriter, r *http.Request) {
	decoder := json.NewDecoder(r.Body)
	var proceso Proceso

	err := decoder.Decode(&proceso)
	if err != nil {
		h.Log.Error("Error al decodificar la RTA del Proceso",
			slog.Attr{Key: "error", Value: slog.StringValue(err.Error())},
		)
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte("Error al decodificar la RTA del Proceso"))
	}

	h.Log.Debug("Me llego la RTA del Proceso",
		slog.Attr{Key: "proceso", Value: slog.AnyValue(proceso)},
	)

	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("ok"))
}
