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

// EnviarProceso envia un proceso al Cpu
func EnviarProceso(w http.ResponseWriter, r *http.Request) {
	// Creo un proceso
	proceso := Proceso{
		ID: 1,
	}

	// Convierto la estructura del proceso a un []bytes (formato en el que se envían las peticiones)
	body, err := json.Marshal(proceso)
	if err != nil {
		log.Printf("error codificando mensaje: %s", err.Error())
	}

	url := fmt.Sprintf("http://%s:%d/procesos", identificacionCPU["ip"].(string), identificacionCPU["puerto"].(int))
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(body))
	if err != nil {
		log.Printf("error enviando mensaje a ip:%s puerto:%d", identificacionCPU["ip"].(string), identificacionCPU["puerto"].(int))
	}

	log.Printf("respuesta del CPU: %s", resp.Status)

	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("ok"))
}

func RespuestaProcesoCPU(w http.ResponseWriter, r *http.Request) {
	decoder := json.NewDecoder(r.Body)
	var proceso Proceso

	err := decoder.Decode(&proceso)
	if err != nil {
		log.Printf("Error al decodificar la RTA del Proceso: %s\n", err.Error())
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("Error al decodificar la RTA del Proceso"))
		return
	}

	log.Println("Me llego la RTA del Proceso")
	log.Printf("%+v\n", proceso)

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("ok"))
}
