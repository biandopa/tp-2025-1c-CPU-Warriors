package main

import (
	"fmt"
	"net/http"
	"os"

	"github.com/sisoputnfrba/tp-golang/kernel/cmd/api"
)

const (
	configFilePath = "./configs/"
)

func main() {
	// El primer argumento es el archivo de configuración y el segundo es el tamaño del proceso
	if len(os.Args) < 4 {
		fmt.Println("Faltan argumentos. Uso: go run kernel.go <archivo_nombre> <tamanio_proceso> <config_id>")
		os.Exit(1)
	}

	archivoNombre := os.Args[1]
	tamanioProceso := os.Args[2] // Tamaño en bytes
	configID := os.Args[3]       // ID de configuración

	configFile := configFilePath + configID + ".json"

	h := api.NewHandler(configFile)
	mux := http.NewServeMux()

	mux.HandleFunc("/io/conexion-inicial", h.ConexionInicialIO)    //IO LISTA --> Kernel
	mux.HandleFunc("/io/desconexion", h.DesconexionIO)             //IO --> Kernel (Notifica desconexión)
	mux.HandleFunc("/cpu/conexion-inicial", h.ConexionInicialCPU)  // CPU  --> Kernel (Envia IP, puerto e ID)  HANDSHAKE
	mux.HandleFunc("/io/peticion-finalizada", h.TerminoPeticionIO) // IO --> KERNEL (usleep)

	mux.HandleFunc("/cpu/proceso", h.RespuestaProcesoCPU) //CPU --> Kernel (Recibe respuesta del proceso de la CPU) PROCESO

	// Kernel --> Memoria
	h.EjecutarPlanificadores(archivoNombre, tamanioProceso)

	err := http.ListenAndServe(fmt.Sprintf(":%d", h.Config.PortKernel), mux)
	if err != nil {
		panic(err)
	}
}
