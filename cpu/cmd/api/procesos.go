package api

import (
	"encoding/json"
	"net/http"

	"github.com/sisoputnfrba/tp-golang/utils/log"
)

func (h *Handler) RecibirProcesos(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	// Leer el cuerpo de la solicitud
	decoder := json.NewDecoder(r.Body)
	proceso := &Proceso{}

	h.Log.Debug("Recibi el proceso")

	// Guarda el valor del body en la variable paquete
	err := decoder.Decode(&proceso)
	if err != nil {
		h.Log.ErrorContext(ctx, "Error al decodificar mensaje.", log.ErrAttr(err))
		http.Error(w, "error al decodificar mensaje", http.StatusInternalServerError)
		return
	}

	h.Log.DebugContext(ctx, "Me llego la peticion del Kernel",
		log.AnyAttr("paquete", proceso),
	)

	// TODO: Devolver PID y PC al Kernel luego de ejecutar
	newPC := h.Ciclo(proceso)

	respose := Proceso{
		PID: proceso.PID,
		PC:  newPC,
	}

	// Conviero la estructura del proceso a un []bytes (formato en el que se envían las peticiones)
	body, _ := json.Marshal(respose)

	// Agrego el status Code 200 a la respuesta
	w.WriteHeader(http.StatusOK)

	// Envío la respuesta al cliente con un mensaje de éxito
	_, _ = w.Write(body)
}
