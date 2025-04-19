package main

import (
	"fmt"
	"net/http"
	"os"

	"github.com/sisoputnfrba/tp-golang/kernel/internal"
)

const (
	configFilePath = "./configs/config.json"
)

func main() {
	h := internal.NewHandler(configFilePath)

	if len(os.Args) < 2 {
		h.Log.Error(fmt.Sprintf("Faltan %d argumentos.", len(os.Args)))
	}

	internal.ArchivoNombre = os.Args[1]
	internal.TamanioProceso = os.Args[2]

	//IO --> Kernel  (le enviará su nombre, ip y puerto) HANDSHAKE
	h.ConeccionInicial()

	mux := http.NewServeMux()

	mux.HandleFunc("/ioConeccionInicial", h.ConeccionInicialIO)   //IO LISTA --> Kernel
	mux.HandleFunc("/cpuConeccionInicial", h.ConeccionInicialCPU) // CPU  --> Kernel (Envia IP, puerto e ID)  HANDSHAKE
	mux.HandleFunc("/ioTerminoPeticion", h.TerminoPeticionIO)     // IO --> KERNEL (usleep)

	mux.HandleFunc("/recibo-proceso-cpu", h.RespuestaProcesoCPU) //CPU --> Kernel (Recibe respuesta del proceso de la CPU) PROCESO

	//mux.HandleFunc("/interrupciones", .RecibirInterrupciones) // Kernel --> CPU Procesos a ejecutar

	err := http.ListenAndServe(fmt.Sprintf(":%d", internal.ClientConfig.PortKernel), mux)
	if err != nil {
		panic(err)
	}
}
