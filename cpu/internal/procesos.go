package internal

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
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
		h.Log.Error("error al decodificar mensaje: %s\n", err.Error())
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte("error al decodificar mensaje"))
		return
	}

	// Agrego el status Code 200 a la respuesta
	w.WriteHeader(http.StatusOK)

	// Envío la respuesta al cliente con un mensaje de éxito
	_, _ = w.Write([]byte("ok"))
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
		log.Printf("error codificando mensaje: %s", err.Error())
	}

	// TODO: Agregar endpoint del Kernel
	url := fmt.Sprintf("http://%s:%d/recibo-proceso-cpu", h.Config.IpKernel, h.Config.PortKernel)
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(body))
	if err != nil {
		log.Printf("error enviando mensaje a ip:%s puerto:%d", h.Config.IpKernel, h.Config.PortKernel)
	}

	if resp != nil {
		log.Printf("respuesta del servidor: %s", resp.Status)
	}

	// Agrego el status Code 200 a la respuesta
	w.WriteHeader(http.StatusOK)

	// Envío la respuesta al cliente con un mensaje de éxito
	_, _ = w.Write([]byte("ok"))
}
