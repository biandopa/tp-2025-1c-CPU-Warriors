package internal

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
)

type IOIdentificacion struct {
	Nombre string `json:"nombre"`
	IP     string `json:"ip"`
	Puerto int    `json:"puerto"`
}
type Config struct {
	IpMemory              string `json:"ip_memory"`
	PortMemory            int    `json:"port_memory"`
	IpKernel              string `json:"ip_kernel"`
	PortKernel            int    `json:"port_kernel"`
	SchedulerAlgorithm    string `json:"scheduler_algorithm"`
	ReadyIngressAlgorithm int    `json:"ready_ingress_algorithm"`
	Alpha                 int    `json:"alpha"`
	SuspensionTime        int    `json:"suspension_time"`
	LogLevel              string `json:"log_level"`
}

var ClientConfig *Config
var ArchivoNombre string
var TamanioProceso string

// TODO: HACER UNA LISTA DE IO
var ioIdentificacion IOIdentificacion

var identificacionCPU = map[string]interface{}{
	"ip":     "",
	"puerto": "",
	"id":     "",
}

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

func ConeccionInicial() {
	log.Printf("Connecion Inicial - archivo: %s, tamaño: %s, config: %+v", ArchivoNombre, TamanioProceso, ClientConfig)

	body, err := json.Marshal(TamanioProceso)
	if err != nil {
		log.Printf("error codificando nombre: %s", err.Error())
		return
	}

	url := fmt.Sprintf("http://%s:%d/pedir-acceso", ClientConfig.Ip_memory, ClientConfig.Port_memory)
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(body))
	if err != nil {
		log.Printf("error enviando mensaje a ip:%s puerto:%d", ClientConfig.Ip_memory, ClientConfig.Port_memory)
	}

	if resp != nil {
		log.Printf("respuesta del servidor: %s", resp.Status)
	} else {
		log.Printf("respuesta del servidor: nil")
	}
}

func ConeccionInicialIO(w http.ResponseWriter, r *http.Request) {
	decoder := json.NewDecoder(r.Body)
	err := decoder.Decode(&ioIdentificacion)
	if err != nil {
		log.Printf("Error al decodificar ioIdentificacion: %s\n", err.Error())
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte("Error al decodificar ioIdentificacion"))
		return
	}

	log.Println("Me llego la conexion de un IO")
	log.Printf("%+v\n", ioIdentificacion)

	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("ok"))
}

func ConeccionInicialCPU(w http.ResponseWriter, r *http.Request) {
	decoder := json.NewDecoder(r.Body)
	err := decoder.Decode(&identificacionCPU)
	if err != nil {
		log.Printf("Error al decodificar ioIdentificacion: %s\n", err.Error())
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte("Error al decodificar ioIdentificacion"))
		return
	}

	log.Println("Me llego la conexion de una CPU")
	log.Printf("%+v\n", identificacionCPU)

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("ok"))
}

// TODO: usarla donde sea necesario
func EnviarPeticionAIO(tiempoSleep int) {

	body, err := json.Marshal(tiempoSleep)
	if err != nil {
		log.Printf("error codificando nombre: %s", err.Error())
		return
	}

	url := fmt.Sprintf("http://%s:%d/petiocionKernel", ioIdentificacion.IP, ioIdentificacion.Puerto)
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(body))
	if err != nil {
		log.Printf("error enviando mensaje a ip:%s puerto:%d", ClientConfig.Ip_kernel, ClientConfig.Port_kernel)
	}

	log.Printf("respuesta del servidor: %s", resp.Status)
}

func TerminoPeticionIO(w http.ResponseWriter, r *http.Request) {

	decoder := json.NewDecoder(r.Body)
	var ioIdentificacionPeticion IOIdentificacion
	err := decoder.Decode(&ioIdentificacionPeticion)
	if err != nil {
		log.Printf("Error al decodificar ioIdentificacion: %s\n", err.Error())
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("Error al decodificar ioIdentificacion"))
		return
	}

	//TODO: Buscar en la lista de ioIdentificacion y cambiarle es status
	log.Println("Me llego la peticion Finalizada de IO")
	log.Printf("%+v\n", ioIdentificacion)

	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("ok"))
}
