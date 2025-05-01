package api

import (
	"encoding/json"
	"net/http"

	"github.com/sisoputnfrba/tp-golang/utils/log"
)

type EspacioDisponible struct {
	Mensaje string `json:"mensaje"`
	Tamaño  int    `json:"tamaño"`
}

// ConsultarEspacioDisponible recibe una consulta sobre el espacio libre en memoria.
// En caso de que haya espacio, se responde con un mensaje de éxito y el tamaño disponible.
// En caso contrario, se responde con un mensaje de error.
// Por el momento, solo responde una respuesta mockeada.
func (h *Handler) ConsultarEspacioDisponible(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Simulamos una consulta al espacio disponible
	espacioDisponible := 1024 // Simulamos que hay 1024 bytes disponibles

	// Enviamos la respuesta al kernel
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	response := EspacioDisponible{
		Mensaje: "Espacio disponible en memoria",
		Tamaño:  espacioDisponible,
	}
	if err := json.NewEncoder(w).Encode(response); err != nil {
		h.Log.ErrorContext(ctx, "Error al codificar la respuesta", log.ErrAttr(err))
		http.Error(w, "Error al codificar la respuesta", http.StatusInternalServerError)
		return
	}

	h.Log.InfoContext(ctx, "Consulta de espacio disponible respondida con éxito",
		log.IntAttr("tamaño_disponible", espacioDisponible),
		log.StringAttr("mensaje", response.Mensaje),
	)

	w.WriteHeader(http.StatusOK)
}
