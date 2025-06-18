package api

import (
	"encoding/json"
	"net/http"

	"github.com/sisoputnfrba/tp-golang/utils/log"
)

func (h *Handler) EnviarInstruccion(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	// Creo instrucciones mockeadas
	instruccion := []Instruccion{
		{
			Instruccion: "NOOP",
		},
		{
			Instruccion: "WRITE",
			Parametros:  []string{"100", "42"},
		},
		{
			Instruccion: "READ",
			Parametros:  []string{"100", "4"},
		},
		{
			Instruccion: "GOTO",
			Parametros:  []string{"3"},
		},
		{
			Instruccion: "EXIT",
		},
	}

	// Conviero la estructura del proceso a un []bytes (formato en el que se envían las peticiones)
	body, err := json.Marshal(instruccion[0]) // Enviamos solo la primera instrucción como ejemplo
	if err != nil {
		h.Log.ErrorContext(ctx, "Error codificando mensaje", log.ErrAttr(err))
		http.Error(w, "Error codificando mensaje", http.StatusBadRequest)
		return
	}

	// Agrego el status Code 200 a la respuesta
	w.WriteHeader(http.StatusOK)

	// Envío la respuesta al cliente con un mensaje de éxito
	_, _ = w.Write(body)
}

func (h *Handler) RecibirInstruccion(w http.ResponseWriter, r *http.Request) {
	// Decode the request body
	var instruccion map[string]interface{}
	err := json.NewDecoder(r.Body).Decode(&instruccion)
	if err != nil {
		h.Log.Error("Error decoding request body", log.ErrAttr(err))
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	h.Log.Info("Instrucción recibida con éxito",
		log.AnyAttr("instruccion", instruccion),
	)

	var i = h.Instrucciones[0] // Simulamos que tomamos la primera instrucción
	body, _ := json.Marshal(i)

	// Respond with success
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(body)
}
