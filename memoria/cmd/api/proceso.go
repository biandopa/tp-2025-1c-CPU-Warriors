package api

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"
)

func (h *Handler) FinalizarProceso(w http.ResponseWriter, r *http.Request) {
	var (
		// Leer PID del endpoint /kernel/fin-proceso/{pid}
		pid = chi.URLParam(r, "pid")
	)

	// Inserte función para finalizar el proceso en memoria
	// Si no se puede finalizar, responde con un error

	w.WriteHeader(http.StatusOK)
	response := map[string]string{
		"message": fmt.Sprintf("Proceso %s finalizado con éxito", pid),
	}
	jsonResponse, _ := json.Marshal(response)
	_, _ = w.Write(jsonResponse)
}
