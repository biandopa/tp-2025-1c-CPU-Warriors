package api

import (
	"encoding/json"
	"net/http"

	"github.com/sisoputnfrba/tp-golang/utils/log"
)

func (h *Handler) RecibirProceso(w http.ResponseWriter, r *http.Request) {
	var (
		// Leer tamanioProceso del queryparameter
		tamanioProceso = r.URL.Query().Get("tamanioProceso")
	)

	if tamanioProceso == "" {
		h.Log.Error("Tamaño del Proceso no proporcionado")
		http.Error(w, "tamaño del oroceso no proporcionado", http.StatusBadRequest)
		return
	}

	// Decode the request body
	var peticion map[string]interface{}
	err := json.NewDecoder(r.Body).Decode(&peticion)
	if err != nil {
		h.Log.Error("Error decoding request body", "error", err)
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	h.Log.Info("Petición recibida con éxito",
		log.AnyAttr("peticion", peticion),
	)

	// Verifica si hay suficiente espacio
	// Inserte función para verificar el espacio disponible

	// Si no hay suficiente espacio, responde con un error
	// Caso contrario, continúa con el procesamiento

	// Respond with success
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("request processed successfully"))
}
