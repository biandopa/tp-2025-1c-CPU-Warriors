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

	if len(os.Args) < 2 {
		h.Log.Error(fmt.Sprintf("Faltan %d argumentos.", len(os.Args)))
		panic("Faltan argumentos para inicializar el módulo Kernel.")
	}

	archivoNombre := os.Args[1]
	tamanioProceso := os.Args[2] // Tamaño en bytes

	// Kernel --> Memoria
	h.EnviarProceso(archivoNombre, tamanioProceso, os.Args[3])

	mux := http.NewServeMux()

	mux.HandleFunc("/io/conexion-inicial", h.ConexionInicialIO)    //IO LISTA --> Kernel
	mux.HandleFunc("/cpu/conexion-inicial", h.ConexionInicialCPU)  // CPU  --> Kernel (Envia IP, puerto e ID)  HANDSHAKE
	mux.HandleFunc("/io/peticion-finalizada", h.TerminoPeticionIO) // IO --> KERNEL (usleep)

	mux.HandleFunc("/cpu/proceso", h.RespuestaProcesoCPU) //CPU --> Kernel (Recibe respuesta del proceso de la CPU) PROCESO

	//mux.HandleFunc("/interrupciones", .RecibirInterrupciones) // Kernel --> CPU Procesos a ejecutar

	err := http.ListenAndServe(fmt.Sprintf(":%d", h.Config.PortKernel), mux)
	if err != nil {
		panic(err)
	}
}
