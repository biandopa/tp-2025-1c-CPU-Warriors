package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/sisoputnfrba/tp-golang/kernel/internal"
)

func main() {

	if len(os.Args) < 2 {
		log.Fatalf("Faltan %d argumentos.", len(os.Args))
	}

	internal.ArchivoNombre = os.Args[1]
	internal.TamanioProceso = os.Args[2]
	internal.ClientConfig = internal.IniciarConfiguracion("config.json")

	//IO --> Kernel  (le enviará su nombre, ip y puerto) HANDSHAKE
	internal.ConeccionInicial()

	mux := http.NewServeMux()

	mux.HandleFunc("/ioConeccionInicial", internal.ConeccionInicialIO)   //IO LISTA --> Kernel
	mux.HandleFunc("/cpuConeccionInicial", internal.ConeccionInicialCPU) // CPU  --> Kernel (Envia IP, puerto e ID)  HANDSHAKE
	mux.HandleFunc("/ioTerminoPeticion", internal.TerminoPeticionIO)     // IO --> KERNEL (usleep)

	mux.HandleFunc("/recibo-proceso-cpu", internal.RespuestaProcesoCPU) //CPU --> Kernel (Recibe respuesta del proceso de la CPU) PROCESO

	//mux.HandleFunc("/interrupciones", .RecibirInterrupciones) // Kernel --> CPU Procesos a ejecutar

	err := http.ListenAndServe(fmt.Sprintf(":%d", internal.ClientConfig.PortKernel), mux)
	if err != nil {
		panic(err)
	}
}
