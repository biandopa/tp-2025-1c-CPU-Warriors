package api

import (
	"encoding/json"
	"log/slog"
	"net/http"
)

func (h *Handler) RecibirProceso(w http.ResponseWriter, r *http.Request) {
	var (
		// Leer tamanioProceso del queryparameter
		tamanioProceso = r.URL.Query().Get("tamanioProceso")
	)

	// Decode the request body
	var peticion map[string]interface{}
	err := json.NewDecoder(r.Body).Decode(&peticion)
	if err != nil {
		h.Log.Error("Error decoding request body", "error", err)
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	h.Log.Info("Petición recibida con éxito",
		slog.Attr{Key: "petición", Value: slog.AnyValue(peticion)},
	)

	// Verifica si hay suficiente espacio
	// Inserte función para verificar el espacio disponible

	// Si no hay suficiente espacio, responde con un error
	// Caso contrario, continúa con el procesamiento

	// Respond with success
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("request processed successfully"))
}
