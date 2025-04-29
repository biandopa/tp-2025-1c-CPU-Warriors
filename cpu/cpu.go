package main

import (
	"fmt"
	"net/http"

	"github.com/sisoputnfrba/tp-golang/cpu/cmd/api"
	"github.com/sisoputnfrba/tp-golang/utils/log"
)

const (
	configFilePath = "./configs/config.json"
)

func main() {
	mux := http.NewServeMux()
	h := api.NewHandler(configFilePath)

	// Recepción de valores
	mux.HandleFunc("POST /memoria/instrucciones", h.RecibirInstrucciones)  // Memoria --> CPU
	mux.HandleFunc("POST /kernel/procesos", h.RecibirProcesos)             // Kernel --> CPU
	mux.HandleFunc("POST /kernel/interrupciones", h.RecibirInterrupciones) // Kernel --> CPU

	// Envío de valores
	mux.HandleFunc("POST /kernel/proceso", h.EnviarProceso)               // CPU --> Kernel
	mux.HandleFunc("POST /kernel/identificacion", h.EnviarIdentificacion) // CPU --> Kernel
	mux.HandleFunc("POST /memoria/instruccion", h.EnviarInstruccion)      // CPU --> Memoria

	cpuAddress := fmt.Sprintf("%s:%d", h.Config.IpCpu, h.Config.PortCpu)
	if err := http.ListenAndServe(cpuAddress, mux); err != nil {
		h.Log.Error("Error starting server", log.ErrAttr(err))
		panic(err)
	}
}
