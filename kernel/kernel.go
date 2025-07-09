package main

import (
	"fmt"
	"net/http"
	"os"

	"github.com/sisoputnfrba/tp-golang/kernel/cmd/api"
)

const (
	configFilePath = "./configs/config.json"
)

func main() {
	h := api.NewHandler(configFilePath)

	// mdilauro: cambio de 2 a 3 porque el primer argumento es el nombre del programa
	// el segundo es el archivo de configuración y el tercero es el tamaño del proceso
	if len(os.Args) < 3 {
		h.Log.Error(fmt.Sprintf("Faltan %d argumentos.", len(os.Args)))
		panic("Faltan argumentos para inicializar el módulo Kernel.")
	}

	archivoNombre := os.Args[1]
	tamanioProceso := os.Args[2] // Tamaño en bytes

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
