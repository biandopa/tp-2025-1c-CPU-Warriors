package main

import (
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
	mux.HandleFunc("/proceso", h.EnviarProceso) // CPU --> Kernel

	if err := http.ListenAndServe(":8080", mux); err != nil {
		h.Log.Error("Error starting server", "err", err)
		panic(err)
	}
}
