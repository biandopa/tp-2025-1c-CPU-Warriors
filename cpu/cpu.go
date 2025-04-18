package main

import (
	"fmt"
	"net/http"

	"github.com/sisoputnfrba/tp-golang/cpu/internal"
)

func main() {
	mux := http.NewServeMux()
	h := internal.NewHandler()

	// Recepción de valores
	mux.HandleFunc("/instrucciones", h.RecibirInstrucciones)   // Memoria --> CPU
	mux.HandleFunc("/procesos", h.RecibirProcesos)             // Kernel --> CPU
	mux.HandleFunc("/interrupciones", h.RecibirInterrupciones) // Kernel --> CPU

	// Envío de valores
	mux.HandleFunc("/enviar-proceso", h.EnviarProceso)         // CPU --> Kernel
	mux.HandleFunc("/envio-ip", h.EnviarIdentificacion)        // CPU --> Kernel
	mux.HandleFunc("/enviar-instruccion", h.EnviarInstruccion) // CPU --> Memoria

	cpuAddress := fmt.Sprintf("%s:%d", h.Config.IpCpu, h.Config.PortCpu)
	if err := http.ListenAndServe(cpuAddress, mux); err != nil {
		h.Log.Error("Error starting server", "err", err)
		panic(err)
	}
}
