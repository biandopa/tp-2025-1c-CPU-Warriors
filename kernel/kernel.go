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

	archivoNombre := os.Args[1]
	tamanioProceso := os.Args[2]

	internal.ClientConfig = internal.IniciarConfiguracion("config.json")

	//IO --> Kernel  (le enviará su nombre, ip y puerto)  HANDSHAKE
	internal.ConeccionInicial(archivoNombre, tamanioProceso, internal.ClientConfig)

	mux := http.NewServeMux()

	// TODO: Los endpoints deben escribirse en minúscula y con guiones (o separar palabras con /)
	// TODO: Si envía body en la petición, se debe usar el método POST
	mux.HandleFunc("/ioConeccionInicial", internal.ConeccionInicialIO)
	mux.HandleFunc("/cpuConeccionInicial", internal.ConeccionInicialCPU)

	err := http.ListenAndServe(fmt.Sprintf(":%d", internal.ClientConfig.PortKernel), mux)
	if err != nil {
		panic(err)
	}

	// Kernel --> Cpu notificaciones de interrupciones LISTO INTERRUPCION
	// Kernel --> Cpu Procesos a ejecutar  LISTO  PROCESO
	// Kernel --> Memoria
	// Kernel --> IO (usleep)?
}
