package main

import (
	"net/http"

	"github.com/sisoputnfrba/tp-golang/cpu/internal"
	"github.com/sisoputnfrba/tp-golang/utils/log"
)

func main() {
	logger := log.BuildLogger()

	handler := http.NewServeMux()

	// Recepción de valores
	handler.HandleFunc("/instrucciones", internal.RecibirInstrucciones)   // Memoria --> CPU
	handler.HandleFunc("/procesos", internal.RecibirProcesos)             // Kernel --> CPU
	handler.HandleFunc("/interrupciones", internal.RecibirInterrupciones) // Kernel --> CPU

	// Envío de valores
	handler.HandleFunc("/proceso", internal.EnviarProceso) // CPU --> Kernel

	if err := http.ListenAndServe(":8080", handler); err != nil {
		logger.Error("Error starting server", "err", err)
		panic(err)
	}
}
