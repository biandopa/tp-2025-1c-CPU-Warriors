package api

import (
	"encoding/json"
	"net/http"

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

	// Ejecutar el proceso en este CPU
	msg := h.Ciclo(&proceso)

	// Enviar respuesta con el nuevo PC
	response := map[string]interface{}{
		"pid":    proceso.PID,
		"pc":     proceso.PC,
		"motivo": msg,
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

// RecibirInterrupcion maneja la recepción de interrupciones del kernel
func (h *Handler) RecibirInterrupcion(w http.ResponseWriter, r *http.Request) {
	var interrupcion internal.Interrupcion
	if err := json.NewDecoder(r.Body).Decode(&interrupcion); err != nil {
		h.Log.Error("Error decodificando interrupción",
			log.ErrAttr(err))
		http.Error(w, "Error decodificando interrupción", http.StatusBadRequest)
		return
	}

	h.Log.Debug("Interrupción recibida del kernel",
		log.IntAttr("pid", interrupcion.PID),
		log.StringAttr("tipo", string(interrupcion.Tipo)),
		log.AnyAttr("enmascarable", interrupcion.EsEnmascarable))

	// Agregar la interrupción a la cola
	h.Service.AgregarInterrupcion(interrupcion)

	// Responder con éxito
	w.Header().Set("Content-Type", "application/json")
	response := map[string]string{"msg": "ok"}
	if err := json.NewEncoder(w).Encode(response); err != nil {
		h.Log.Error("Error codificando respuesta de interrupción",
			log.ErrAttr(err))
		http.Error(w, "Error interno del servidor", http.StatusInternalServerError)
		return
	}
}
