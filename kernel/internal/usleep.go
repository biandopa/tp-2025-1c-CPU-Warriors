package internal

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
)

type Usleep struct {
	PID         int `json:"pid"`
	TiempoSleep int `json:"tiempo_sleep"`
}

// EnviarUsleep envia un usleep al IO
func SendUsleep(puertoIO, iPIO, nombre string, timeSleep int) error {
	// Crear el JSON con los datos necesarios
	usleep := &Usleep{
		PID:         0, // Cambiar por el PID correspondiente
		TiempoSleep: timeSleep,
	}

	jsonData, err := json.Marshal(usleep)
	if err != nil {
		return err
	}

	// Realizar la petición POST al IO
	url := fmt.Sprintf("http://%s:%s/kernel/usleep", puertoIO, iPIO)
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return err
	}

	// TODO: Ver qué hacer con la respuesta
	if resp != nil {
		defer func() {
			if err = resp.Body.Close(); err != nil {
				fmt.Println("Error cerrando el cuerpo de la respuesta:", err)
			}
		}()
	}

	return nil
}
