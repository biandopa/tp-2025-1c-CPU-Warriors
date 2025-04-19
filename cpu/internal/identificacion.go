package internal

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/sisoputnfrba/tp-golang/utils/log"
)

func (h *Handler) EnviarIdentificacion(w http.ResponseWriter, r *http.Request) {
	identificacion := map[string]interface{}{
		"ip":     h.Config.IpCpu,
		"puerto": h.Config.PortCpu,
		"id":     "cpu-id", // Cambiar por el ID real
	}

	// Convierto la estructura del proceso a un []bytes (formato en el que se envían las peticiones)
	body, err := json.Marshal(identificacion)
	if err != nil {
		h.Log.Error("Error codificando mensaje", log.ErrAttr(err))
		http.Error(w, "error codificando mensaje", http.StatusInternalServerError)
		return
	}

	// TODO: Agregar endpoint del Kernel
	url := fmt.Sprintf("http://%s:%d/cpuConeccionInicial", h.Config.IpKernel, h.Config.PortKernel)
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(body))
	if err != nil {
		h.Log.Error("Error enviando identificacion a Kernel",
			slog.Attr{Key: "ip", Value: slog.StringValue(h.Config.IpKernel)},
			slog.Attr{Key: "puerto", Value: slog.IntValue(h.Config.PortKernel)},
			log.ErrAttr(err),
		)
		http.Error(w, "Error enviando identificacion", http.StatusBadRequest)
		return
	}

	if resp != nil {
		h.Log.Info("Respuesta del servidor",
			slog.Attr{Key: "status", Value: slog.StringValue(resp.Status)},
			slog.Attr{Key: "body", Value: slog.StringValue(string(body))},
		)
	} else {
		h.Log.Info("Respuesta del servidor: nil")
	}

	// Agrego el status Code 200 a la respuesta
	w.WriteHeader(http.StatusOK)

	// Envío la respuesta al cliente con un mensaje de éxito
	_, _ = w.Write([]byte("ok"))
}
