package internal

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
)

type IOIdentificacion struct {
	Nombre string `json:"nombre"`
	IP     string `json:"ip"`
	Puerto int    `json:"puerto"`
}

type Config struct {
	IpKernel   string `json:"ip_kernel"`
	PortKernel int    `json:"port_kernel"`
	PortIo     int    `json:"port_io"`
	IpIo       string `json:"ip_io"`
	LogLevel   string `json:"log_level"`
}

var ClientConfig *Config

func IniciarConfiguracion(filePath string) *Config {
	var config *Config
	configFile, err := os.Open(filePath)
	if err != nil {
		log.Fatal(err.Error())
	}
	defer func() {
		_ = configFile.Close()
	}()

	jsonParser := json.NewDecoder(configFile)
	if err = jsonParser.Decode(&config); err != nil {
		log.Fatal(err.Error())
	}

	return config
}

func ConeccionInicial(nombre string, ClientConfig1 *Config) {
	data := IOIdentificacion{
		Nombre: nombre,
		IP:     ClientConfig1.IpIo,
		Puerto: ClientConfig1.PortIo,
	}

	body, err := json.Marshal(data)
	if err != nil {
		log.Printf("error codificando nombre: %s", err.Error())
		return
	}

	url := fmt.Sprintf("http://%s:%d/ioConeccionInicial", ClientConfig1.IpKernel, ClientConfig1.PortKernel)
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(body))
	if err != nil {
		log.Printf("error enviando mensaje a ip:%s puerto:%d", ClientConfig1.IpKernel, ClientConfig1.PortKernel)
	}

	if resp != nil {
		log.Printf("respuesta del servidor: %s", resp.Status)
	} else {
		log.Printf("No se recibió respuesta del servidor")
	}
}
