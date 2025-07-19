package api

import (
	"encoding/json"
	"fmt"
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
	var proceso Proceso
	err := json.NewDecoder(r.Body).Decode(&proceso)
	if err != nil {
		h.Log.Error("Error decoding request body", log.ErrAttr(err))
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	h.Log.Debug("Petición de envío de instrucción recibida con éxito",
		log.AnyAttr("proceso", proceso),
	)

	h.mutexInstrucciones.RLock()
	defer h.mutexInstrucciones.RUnlock()
	// Verificamos si el proceso tiene almacenadas instrucciones
	if _, exists := h.Instrucciones[proceso.PID]; !exists {
		h.Log.Debug("No hay instrucciones almacenadas para el proceso",
			log.IntAttr("pid", proceso.PID),
			log.IntAttr("pc", proceso.PC),
		)
		http.Error(w, "no instructions available for the process", http.StatusBadRequest)
		return
	}

	// Si tuvo, pero no quedan más instrucciones, devolvemos un status 204
	if len(h.Instrucciones[proceso.PID]) == 0 {
		h.Log.Debug("No quedan más instrucciones para el proceso",
			log.IntAttr("pid", proceso.PID),
			log.IntAttr("pc", proceso.PC),
		)
		http.Error(w, "no more instructions for the process", http.StatusNoContent)
		return
	}

	instruccion := h.Instrucciones[proceso.PID][proceso.PC]

	/* Log obligatorio: Obtener instrucción
	“## PID: <PID> - Obtener instrucción: <PC> - Instrucción: <INSTRUCCIÓN> <...ARGS>”*/
	h.Log.Info(fmt.Sprintf("## PID: %d - Obtener instrucción: %d - Instrucción: %s",
		proceso.PID, proceso.PC, instruccion))

	// Leemos la instrucción asociada al proceso. Usamos el PC como index del array y luego la enviamos al cliente
	body, _ := json.Marshal(instruccion)

	// Respond with success
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(body)
}
