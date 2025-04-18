package internal

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/sisoputnfrba/tp-golang/utils/log"
)

func (h *Handler) EnviarInstrucciones(w http.ResponseWriter, r *http.Request) {
	// Creo instruccion
	instruccion := map[string]interface{}{
		"tipo": "instruccion",
		"datos": map[string]interface{}{
			"codigo": "codigo de la instruccion",
		},
	}

	// Conviero la estructura del proceso a un []bytes (formato en el que se envían las peticiones)
	body, err := json.Marshal(instruccion)
	if err != nil {
		h.Log.Error("Error codificando mensaje", log.ErrAttr(err))
		http.Error(w, "Error codificando mensaje", http.StatusBadRequest)
		return
	}

	url := fmt.Sprintf("http://%s:%d/instrucciones", h.Config.IpCpu, h.Config.PortCpu)
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(body))
	if err != nil {
		h.Log.Error("Error enviando mensaje",
			slog.Attr{Key: "ip", Value: slog.StringValue(h.Config.IpCpu)},
			slog.Attr{Key: "puerto", Value: slog.IntValue(h.Config.PortCpu)},
			log.ErrAttr(err),
		)
		http.Error(w, "Error enviando mensaje", http.StatusBadRequest)
		return
	}

	if resp != nil {
		defer func() {
			if err = resp.Body.Close(); err != nil {
				h.Log.Error("Error cerrando el cuerpo de la respuesta", log.ErrAttr(err))
			}
		}()

		if resp.StatusCode != http.StatusOK {
			h.Log.Error("Error en la respuesta del CPU",
				slog.Attr{Key: "status_code", Value: slog.IntValue(resp.StatusCode)},
			)
			http.Error(w, "error en la respuesta del CPU", http.StatusInternalServerError)
			return
		}
		h.Log.Info("Mensaje enviado al CPU con éxito",
			slog.Attr{Key: "status_code", Value: slog.IntValue(resp.StatusCode)},
		)
	} else {
		h.Log.Error("Error al enviar mensaje al CPU")
		http.Error(w, "error al enviar mensaje al CPU", http.StatusInternalServerError)
		return
	}

	// Agrego el status Code 200 a la respuesta
	w.WriteHeader(http.StatusOK)

	// Envío la respuesta al cliente con un mensaje de éxito
	_, _ = w.Write([]byte("ok"))
}

func (h *Handler) RecibirInstruccion(w http.ResponseWriter, r *http.Request) {
	// Decode the request body
	var instruccion map[string]interface{}
	err := json.NewDecoder(r.Body).Decode(&instruccion)
	if err != nil {
		h.Log.Error("Error decoding request body", "error", err)
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	h.Log.Info("Instrucción recibida con éxito",
		slog.Attr{Key: "instruccion", Value: slog.AnyValue(instruccion)},
	)

	// Respond with success
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("request processed successfully"))
}
