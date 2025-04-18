package internal

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
)

// Ambos 2 muy probablemente sean una lista cada uno
type IOIdentificacion struct {
	Nombre string `json:"nombre"`
	IP     string `json:"ip"`
	Puerto int    `json:"puerto"`
}

type CPUIdentificacion struct {
	Identificador string `json:"identificador"`
	IP            string `json:"ip"`
	Puerto        int    `json:"puerto"`
}

type Config struct {
	Ip_memory               string `json:"ip_memory"`
	Port_memory             int    `json:"port_memory"`
	Ip_kernel               string `json:"ip_kernel"`
	Port_kernel             int    `json:"port_kernel"`
	Scheduler_algorithm     string `json:"scheduler_algorithm"`
	Ready_ingress_algorithm int    `json:"ready_ingress_algorithm"`
	Alpha                   int    `json:"alpha"`
	Suspension_Time         int    `json:"suspension_time"`
	Log_level               string `json:"log_level"`
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

func ConeccionInicial(archivoNombre string, tamanioProceso string, ClientConfig1 *Config) {

	log.Printf("Connecion Inicial - archivo: %s, tamaño: %s, config: %+v", archivoNombre, tamanioProceso, ClientConfig1)

	body, err := json.Marshal(tamanioProceso)
	if err != nil {
		log.Printf("error codificando nombre: %s", err.Error())
		return
	}

	url := fmt.Sprintf("http://%s:%d/memoriaConeccionInicial", ClientConfig1.Ip_memory, ClientConfig1.Port_memory)
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(body))
	if err != nil {
		log.Printf("error enviando mensaje a ip:%s puerto:%d", ClientConfig1.Ip_memory, ClientConfig1.Port_memory)
	}

	log.Printf("respuesta del servidor: %s", resp.Status)
}

func ConeccionInicialIO(w http.ResponseWriter, r *http.Request) {
	decoder := json.NewDecoder(r.Body)
	var ioIdentificacion IOIdentificacion
	err := decoder.Decode(&ioIdentificacion)
	if err != nil {
		log.Printf("Error al decodificar ioIdentificacion: %s\n", err.Error())
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("Error al decodificar ioIdentificacion"))
		return
	}

	log.Println("Me llego la conexion de un IO")
	log.Printf("%+v\n", ioIdentificacion)

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("ok"))
}

func ConeccionInicialCPU(w http.ResponseWriter, r *http.Request) {
	decoder := json.NewDecoder(r.Body)
	var cpuIdentificacion CPUIdentificacion
	err := decoder.Decode(&cpuIdentificacion)
	if err != nil {
		log.Printf("Error al decodificar ioIdentificacion: %s\n", err.Error())
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("Error al decodificar ioIdentificacion"))
		return
	}

	log.Println("Me llego la conexion de una CPU")
	log.Printf("%+v\n", cpuIdentificacion)

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("ok"))
}
