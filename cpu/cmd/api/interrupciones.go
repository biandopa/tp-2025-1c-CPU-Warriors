package api

import (
	"encoding/json"
	"net/http"

	"github.com/sisoputnfrba/tp-golang/cpu/internal"
	"github.com/sisoputnfrba/tp-golang/utils/log"
)

// RecibirInterrupciones maneja las interrupciones enviadas por el Kernel.
func (h *Handler) RecibirInterrupciones(w http.ResponseWriter, r *http.Request) {
	var (
		ctx          = r.Context()
		interrupcion internal.Interrupcion
	)
	// Leo el cuerpo de la solicitud y guardo el valor del body en la variable interrupcion
	if err := json.NewDecoder(r.Body).Decode(&interrupcion); err != nil {
		h.Log.ErrorContext(ctx, "Error al decodificar interrupción",
			log.ErrAttr(err))
		http.Error(w, "error al decodificar mensaje", http.StatusInternalServerError)
		return
	}

	//Log obligatorio: Interrupción recibida
	//“## Llega interrupción al puerto Interrupt”
	h.Log.DebugContext(ctx, "Recibí interrupciones del Kernel",
		log.AnyAttr("interrupción", interrupcion),
	)

	h.Service.AgregarInterrupcion(interrupcion)

	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("ok"))
}
