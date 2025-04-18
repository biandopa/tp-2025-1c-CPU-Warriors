package internal

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
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
		log.Printf("error codificando mensaje: %s", err.Error())
	}

	url := fmt.Sprintf("http://%s:%d/instrucciones", h.Config.IpCpu, h.Config.PortCpu)
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(body))
	if err != nil {
		log.Printf("error enviando mensaje a ip:%s puerto:%d", h.Config.IpCpu, h.Config.PortCpu)
	}

	if resp != nil {
		log.Printf("respuesta del servidor: %s", resp.Status)
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
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	h.Log.Info("RecibirInstruccion", "instruccion", instruccion)

	// Respond with success
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("Request processed successfully"))
}
