package main

import (
	"fmt"
	"log/slog"
	"net/http"

	"github.com/sisoputnfrba/tp-golang/cpu/internal"
)

func main() {
	mux := http.NewServeMux()
	h := internal.NewHandler()

	// Recepción de valores
	mux.HandleFunc("POST /instrucciones", h.RecibirInstrucciones)   // Memoria --> CPU
	mux.HandleFunc("POST /procesos", h.RecibirProcesos)             // Kernel --> CPU
	mux.HandleFunc("POST /interrupciones", h.RecibirInterrupciones) // Kernel --> CPU

	// Envío de valores
	mux.HandleFunc("POST /enviar-proceso", h.EnviarProceso)         // CPU --> Kernel
	mux.HandleFunc("POST /envio-ip", h.EnviarIdentificacion)        // CPU --> Kernel
	mux.HandleFunc("POST /enviar-instruccion", h.EnviarInstruccion) // CPU --> Memoria

	cpuAddress := fmt.Sprintf("%s:%d", h.Config.IpCpu, h.Config.PortCpu)
	if err := http.ListenAndServe(cpuAddress, mux); err != nil {
		h.Log.Error("Error starting server",
			slog.Attr{Key: "err", Value: slog.StringValue(err.Error())},
		)
		panic(err)
	}
}
