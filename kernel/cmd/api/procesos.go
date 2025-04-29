package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
)

type Proceso struct {
	ID int `json:"id"`
}

// EnviarProceso envia un proceso al Cpu
func (h *Handler) EnviarProceso(w http.ResponseWriter, r *http.Request) {
	// Creo un proceso
	proceso := Proceso{
		ID: 1,
	}

	// Convierto la estructura del proceso a un []bytes (formato en el que se envían las peticiones)
	body, err := json.Marshal(proceso)
	if err != nil {
		h.Log.Error("error codificando mensaje",
			slog.Attr{Key: "error", Value: slog.StringValue(err.Error())},
		)
		http.Error(w, "error codificando mensaje", http.StatusInternalServerError)
	}

	url := fmt.Sprintf("http://%s:%d/procesos", identificacionCPU["ip"].(string), identificacionCPU["puerto"].(int))
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

func (h *Handler) RespuestaProcesoCPU(w http.ResponseWriter, r *http.Request) {
	decoder := json.NewDecoder(r.Body)
	var proceso Proceso

	err := decoder.Decode(&proceso)
	if err != nil {
		h.Log.Error("Error al decodificar la RTA del Proceso",
			slog.Attr{Key: "error", Value: slog.StringValue(err.Error())},
		)
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte("Error al decodificar la RTA del Proceso"))
	}

	h.Log.Debug("Me llego la RTA del Proceso",
		slog.Attr{Key: "proceso", Value: slog.AnyValue(proceso)},
	)

	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("ok"))
}
