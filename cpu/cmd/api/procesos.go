package api

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/sisoputnfrba/tp-golang/utils/log"
)

// RecibirProcesos maneja la recepción de un proceso del kernel
func (h *Handler) RecibirProcesos(w http.ResponseWriter, r *http.Request) {
	var proceso Proceso
	if err := json.NewDecoder(r.Body).Decode(&proceso); err != nil {
		h.Log.Error("Error decodificando proceso",
			log.ErrAttr(err))
		http.Error(w, "Error decodificando proceso", http.StatusBadRequest)
		return
	}

	h.Log.Debug("Proceso recibido del kernel",
		log.IntAttr("pid", proceso.PID),
		log.IntAttr("pc", proceso.PC))

	// Settear el tiempo de inicio del proceso
	h.TiempoInicio = time.Now()

	// Ejecutar el proceso en este CPU
	msg := h.Ciclo(&proceso)

	// Enviar respuesta con el nuevo PC
	response := map[string]interface{}{
		"pid":    proceso.PID,
		"pc":     proceso.PC,
		"motivo": msg,
		"rafaga": time.Since(h.TiempoInicio).Milliseconds(),
	}

	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		h.Log.Error("Error codificando respuesta",
			log.ErrAttr(err))
		http.Error(w, "Error interno del servidor", http.StatusInternalServerError)
		return
	}
}
