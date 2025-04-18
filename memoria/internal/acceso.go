package internal

import (
	"encoding/json"
	"net/http"
)

func (h *Handler) RecibirPeticionAcceso(w http.ResponseWriter, r *http.Request) {
	// Decode the request body
	var peticion map[string]interface{}
	err := json.NewDecoder(r.Body).Decode(&peticion)
	if err != nil {
		h.Log.Error("Error decoding request body", "error", err)
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	h.Log.Info("RecibirPeticionAcceso", "peticion", peticion)

	// Respond with success
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("Request processed successfully"))
}
