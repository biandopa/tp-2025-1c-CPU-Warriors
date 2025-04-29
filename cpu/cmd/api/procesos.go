package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/sisoputnfrba/tp-golang/utils/log"
)

type Proceso struct {
	ID int `json:"id"`
}

func (h *Handler) RecibirProcesos(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	// Leer el cuerpo de la solicitud
	decoder := json.NewDecoder(r.Body)
	paquete := map[string]interface{}{}

	// Guarda el valor del body en la variable paquete
	err := decoder.Decode(&paquete)
	if err != nil {
		h.Log.ErrorContext(ctx, "Error al decodificar mensaje.", log.ErrAttr(err))
		http.Error(w, "error al decodificar mensaje", http.StatusInternalServerError)
		return
	}

	h.Log.DebugContext(ctx, "Me llego la peticion del Kernel",
		log.AnyAttr("paquete", paquete),
	)

	// Agrego el status Code 200 a la respuesta
	w.WriteHeader(http.StatusOK)

	// Envío la respuesta al cliente con un mensaje de éxito
	_, _ = w.Write([]byte("ok"))
}

// EnviarProceso envia un proceso al kernel
func (h *Handler) EnviarProceso(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	// Creo un proceso
	proceso := Proceso{
		ID: 1,
	}

	// Conviero la estructura del proceso a un []bytes (formato en el que se envían las peticiones)
	body, err := json.Marshal(proceso)
	if err != nil {
		h.Log.ErrorContext(ctx, "error codificando mensaje.", log.ErrAttr(err))
		http.Error(w, "error codificando mensaje", http.StatusBadRequest)
	}

	url := fmt.Sprintf("http://%s:%d/cpu/proceso", h.Config.IpKernel, h.Config.PortKernel)
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(body))
	if err != nil {
		h.Log.ErrorContext(ctx, "Error enviando proceso al Kernel",
			log.StringAttr("ip", h.Config.IpKernel),
			log.IntAttr("puerto", h.Config.PortKernel),
			log.ErrAttr(err),
		)
		http.Error(w, "error enviando mensaje", http.StatusBadRequest)
		return
	}

	if resp != nil {
		h.Log.Debug("Respuesta del servidor recibida.",
			slog.Attr{Key: "status", Value: slog.StringValue(resp.Status)},
			slog.Attr{Key: "body", Value: slog.AnyValue(resp.Body)},
		)
	}

	// Agrego el status Code 200 a la respuesta
	w.WriteHeader(http.StatusOK)

	// Envío la respuesta al cliente con un mensaje de éxito
	_, _ = w.Write([]byte("ok"))
}
