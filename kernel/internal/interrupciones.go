package internal

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
)

type Interrupcion struct {
	Interrupcion string `json:"interrupcion"`
}

// EnviarProceso envia una Interrupcion al Cpu
func EnviarInterrupcion(w http.ResponseWriter, r *http.Request) {
	// Creo un proceso
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
		log.Printf("error enviando mensaje a ip:%s puerto:%d", identificacionCPU["ip"].(string), identificacionCPU["puerto"].(int))
	}

	log.Printf("respuesta del CPU: %s", resp.Status)

	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("ok"))
}
