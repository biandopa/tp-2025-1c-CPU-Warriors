package internal

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
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
		log.Printf("error codificando mensaje: %s", err.Error())
	}

	// TODO: Agregar endpoint del Kernel
	url := fmt.Sprintf("http://%s:%d/cpuConeccionInicial", h.Config.IpKernel, h.Config.PortKernel)
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(body))
	if err != nil {
		log.Printf("error enviando mensaje a ip:%s puerto:%d", h.Config.IpKernel, h.Config.PortKernel)
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte("error enviando mensaje"))
	}

	if resp != nil {
		log.Printf("respuesta del servidor: %s", resp.Status)
	}

	// Agrego el status Code 200 a la respuesta
	w.WriteHeader(http.StatusOK)

	// Envío la respuesta al cliente con un mensaje de éxito
	_, _ = w.Write([]byte("ok"))

}
