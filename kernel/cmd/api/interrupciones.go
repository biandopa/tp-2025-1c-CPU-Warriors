package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"log/slog"
	"net/http"
)

type Interrupcion struct {
	Interrupcion string `json:"interrupcion"`
}

// EnviarInterrupcion envia una Interrupcion al Cpu
func (h *Handler) EnviarInterrupcion(w http.ResponseWriter, r *http.Request) {
	// Creo una interrupción
	interrupcion := Interrupcion{
		Interrupcion: "Interrupcion1",
	}

	// Convierto la estructura del proceso a un []bytes (formato en el que se envían las peticiones)
	body, err := json.Marshal(interrupcion)
	if err != nil {
		log.Printf("error codificando mensaje: %s", err.Error())
	}

	url := fmt.Sprintf("http://%s:%d/instrucciones", identificacionCPU["ip"].(string), identificacionCPU["puerto"].(int))
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(body))
	if err != nil {
		h.Log.Error("error enviando mensaje",
			slog.Attr{Key: "error", Value: slog.StringValue(err.Error())},
			slog.Attr{Key: "ip", Value: slog.StringValue(identificacionCPU["ip"].(string))},
			slog.Attr{Key: "puerto", Value: slog.IntValue(identificacionCPU["puerto"].(int))},
		)
		http.Error(w, "error enviando mensaje", http.StatusBadRequest)
		return
	}

	if resp != nil {
		h.Log.Debug("Respuesta del servidor",
			slog.Attr{Key: "status", Value: slog.StringValue(resp.Status)},
			slog.Attr{Key: "body", Value: slog.StringValue(string(body))},
		)
	} else {
		h.Log.Debug("Respuesta del servidor: nil")
	}

	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("ok"))
}
