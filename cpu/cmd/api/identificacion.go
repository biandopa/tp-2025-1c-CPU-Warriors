package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
)

func (h *Handler) EnviarIdentificacion(nombre string) {
	data := map[string]interface{}{
		"ip":     h.Config.IpCpu,
		"puerto": h.Config.PortCpu,
		"id":     nombre, // Cambiar por el ID real
	}

	h.Log.Info("Respuesta del servidor",
		slog.Attr{Key: "ipKernel", Value: slog.StringValue(h.Config.IpKernel)},
		slog.Attr{Key: "portKernel", Value: slog.IntValue(h.Config.PortKernel)},
	)

	body, err := json.Marshal(data)
	if err != nil {
		h.Log.Error("Error al serializar ioIdentificacion",
			slog.Attr{Key: "error", Value: slog.StringValue(err.Error())},
		)
		return
	}

	url := fmt.Sprintf("http://%s:%d/cpu/conexion-inicial", h.Config.IpKernel, h.Config.PortKernel)
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(body))
	if err != nil {
		h.Log.Error("error enviando mensaje",
			slog.Attr{Key: "error", Value: slog.StringValue(err.Error())},
			slog.Attr{Key: "ip", Value: slog.StringValue(h.Config.IpKernel)},
			slog.Attr{Key: "puerto", Value: slog.IntValue(h.Config.PortKernel)},
		)
	}

	if resp != nil {
		h.Log.Info("Respuesta del servidor",
			slog.Attr{Key: "status", Value: slog.StringValue(resp.Status)},
			slog.Attr{Key: "body", Value: slog.StringValue(string(body))},
		)
	} else {
		h.Log.Info("Respuesta del servidor: nil")
	}
	// Convierto la estructura del proceso a un []bytes (formato en el que se envían las peticiones)
	/**body, err := json.Marshal(identificacion)
	if err != nil {
		h.Log.ErrorContext(ctx, "Error codificando mensaje", log.ErrAttr(err))
		http.Error(w, "error codificando mensaje", http.StatusInternalServerError)
		return
	}

	url := fmt.Sprintf("http://%s:%d/cpu/conexion-inicial", h.Config.IpKernel, h.Config.PortKernel)
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(body))
	if err != nil {
		h.Log.ErrorContext(ctx, "Error enviando identificacion a Kernel",
			log.StringAttr("ip", h.Config.IpKernel),
			log.IntAttr("port", h.Config.PortKernel),
			log.ErrAttr(err),
		)
		http.Error(w, "Error enviando identificacion", http.StatusBadRequest)
		return
	}

	if resp != nil {
		h.Log.Info("Respuesta del servidor",
			log.StringAttr("status", resp.Status),
		)
	} else {
		h.Log.Info("Respuesta del servidor: nil")
	}

	// Agrego el status Code 200 a la respuesta
	w.WriteHeader(http.StatusOK)

	// Envío la respuesta al cliente con un mensaje de éxito
	_, _ = w.Write([]byte("ok"))*/
}
