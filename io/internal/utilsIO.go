package internal

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"
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
var NombreIO string

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

func ConeccionInicial() {

	data := IOIdentificacion{
		Nombre: NombreIO,
		IP:     ClientConfig.Ip_io,
		Puerto: ClientConfig.Port_io,
	}

	body, err := json.Marshal(data)
	if err != nil {
		log.Printf("error codificando nombre: %s", err.Error())
		return
	}

	url := fmt.Sprintf("http://%s:%d/ioConeccionInicial", ClientConfig.Ip_kernel, ClientConfig.Port_kernel)
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(body))
	if err != nil {
		log.Printf("error enviando mensaje a ip:%s puerto:%d", ClientConfig.Ip_kernel, ClientConfig.Port_kernel)
	}

	log.Printf("respuesta del servidor: %s", resp.Status)
}

func EjecutarPeticion(w http.ResponseWriter, r *http.Request) {
	decoder := json.NewDecoder(r.Body)
	var tiempoSleep int
	err := decoder.Decode(&tiempoSleep)
	if err != nil {
		log.Printf("Error al decodificar ioIdentificacion: %s\n", err.Error())
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("Error al decodificar ioIdentificacion"))
		return
	}

	log.Println("Me llego la peticion del Kernel")
	log.Printf("%+d\n", tiempoSleep)

	time.Sleep(time.Duration(tiempoSleep) * time.Microsecond)
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("ok"))

	//Avisa que termino
	AvisarAKernelFinalizacionPeticion()
}

func AvisarAKernelFinalizacionPeticion() {

	url := fmt.Sprintf("http://%s:%d/ioTerminoPeticion", ClientConfig.Ip_kernel, ClientConfig.Port_kernel)
	resp, err := http.Post(url, "application/json", bytes.NewBuffer([]byte("{}")))
	if err != nil {
		log.Printf("error enviando mensaje a ip:%s puerto:%d", ClientConfig.Ip_kernel, ClientConfig.Port_kernel)
	}

	log.Printf("respuesta del servidor: %s", resp.Status)
}
