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
	Ip_kernel   string `json:"ip_kernel"`
	Port_kernel int    `json:"port_kernel"`
	Port_io     int    `json:"port_io"`
	Ip_io       string `json:"ip_io"`
	Log_level   string `json:"log_level"`
}

var ClientConfig *Config

func IniciarConfiguracion(filePath string) *Config {
	var config *Config
	configFile, err := os.Open(filePath)
	if err != nil {
		log.Fatal(err.Error())
	}
	defer configFile.Close()

	jsonParser := json.NewDecoder(configFile)
	jsonParser.Decode(&config)

	return config
}

func ConeccionInicial(nombre string, ClientConfig1 *Config) {

	data := IOIdentificacion{
		Nombre: nombre,
		IP:     ClientConfig1.Ip_io,
		Puerto: ClientConfig1.Port_io,
	}

	body, err := json.Marshal(data)
	if err != nil {
		log.Printf("error codificando nombre: %s", err.Error())
		return
	}

	url := fmt.Sprintf("http://%s:%d/ioConeccionInicial", ClientConfig1.Ip_kernel, ClientConfig1.Port_kernel)
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(body))
	if err != nil {
		log.Printf("error enviando mensaje a ip:%s puerto:%d", ClientConfig1.Ip_kernel, ClientConfig1.Port_kernel)
	}

	log.Printf("respuesta del servidor: %s", resp.Status)
}
