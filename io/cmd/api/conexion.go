package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
)

func (h *Handler) ConexionInicialKernel(nombre string) {
	// Estructura para enviar la identificación del IO al kernel
	data := IOIdentificacion{
		Nombre: nombre,
		IP:     h.Config.IpIo,
		Puerto: h.Config.PortIo,
	}

	// Serializar la estructura a JSON
	body, err := json.Marshal(data)
	if err != nil {
		h.Log.Error("Error al serializar ioIdentificacion",
			slog.Attr{Key: "error", Value: slog.StringValue(err.Error())},
		)
		return
	}

	// Enviar la solicitud POST al kernel
	url := fmt.Sprintf("http://%s:%d/io/conexion-inicial", h.Config.IpKernel, h.Config.PortKernel)
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
}
