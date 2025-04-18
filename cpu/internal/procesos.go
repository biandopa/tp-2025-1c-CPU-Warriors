package internal

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
	// Leer el cuerpo de la solicitud
	decoder := json.NewDecoder(r.Body)
	paquete := map[string]interface{}{}

	// Guarda el valor del body en la variable paquete
	err := decoder.Decode(&paquete)
	if err != nil {
		h.Log.Error("Error al decodificar mensaje.", log.ErrAttr(err))
		http.Error(w, "error al decodificar mensaje", http.StatusInternalServerError)
		return
	}

	// Agrego el status Code 200 a la respuesta
	w.WriteHeader(http.StatusOK)

	// Envío la respuesta al cliente con un mensaje de éxito
	_, _ = w.Write([]byte("ok"))
	return
}

// EnviarProceso envia un proceso al kernel
func (h *Handler) EnviarProceso(w http.ResponseWriter, r *http.Request) {
	// Creo un proceso
	proceso := Proceso{
		ID: 1,
	}

	// Conviero la estructura del proceso a un []bytes (formato en el que se envían las peticiones)
	body, err := json.Marshal(proceso)
	if err != nil {
		h.Log.Error("error codificando mensaje.", log.ErrAttr(err))
		http.Error(w, "error codificando mensaje", http.StatusBadRequest)
	}

	// TODO: Agregar endpoint del Kernel
	url := fmt.Sprintf("http://%s:%d/{{endpoint-kernel}}", h.Config.IpKernel, h.Config.PortKernel)
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(body))
	if err != nil {
		h.Log.Error("Error enviando proceso al Kernel",
			slog.Attr{Key: "ip", Value: slog.StringValue(h.Config.IpKernel)},
			slog.Attr{Key: "puerto", Value: slog.IntValue(h.Config.PortKernel)},
			log.ErrAttr(err),
		)
		http.Error(w, "error enviando mensaje", http.StatusBadRequest)
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
