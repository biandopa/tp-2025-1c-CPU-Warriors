package main

import (
	"fmt"
	"net/http"
	"os"

	"github.com/sisoputnfrba/tp-golang/cpu/cmd/api"
	"github.com/sisoputnfrba/tp-golang/utils/log"
)

const (
	configFilePath = "./configs/config.json"
)

func main() {
	mux := http.NewServeMux()
	h := api.NewHandler(configFilePath)

	//para que tome el argumento debe ingresarse asi "go run cpu.go Identificador"
	identificadorCPU := os.Args[1]

	h.Log.Debug("Inicializando interfaz CPU",
		log.StringAttr("nombreCPU", identificadorCPU),
	)

	//IO --> Kernel  (le enviará su nombre, ip y puerto)  HANDSHAKE
	h.EnviarIdentificacion(identificadorCPU)

	// Recepción de valores
	mux.HandleFunc("POST /memoria/instrucciones", h.RecibirInstrucciones)  // Memoria --> CPU
	mux.HandleFunc("POST /kernel/procesos", h.RecibirProcesos)             // Kernel --> CPU
	mux.HandleFunc("POST /kernel/interrupciones", h.RecibirInterrupciones) // Kernel --> CPU

	// Envío de valores
	//mux.HandleFunc("POST /kernel/proceso", h.EnviarProceso) // CPU --> Kernel
	//mux.HandleFunc("POST /kernel/identificacion", h.EnviarIdentificacion) // CPU --> Kernel
	mux.HandleFunc("POST /memoria/instruccion", h.EnviarInstruccion) // CPU --> Memoria

	// Nota: Le pasamos por argumento el puerto para que levante muchas CPUs
	cpuAddress := fmt.Sprintf("%s:%s", h.Config.IpCpu, identificadorCPU)
	if err := http.ListenAndServe(cpuAddress, mux); err != nil {
		h.Log.Error("Error starting server", log.ErrAttr(err))
		panic(err)
	}
}
