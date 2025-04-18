package main

import (
	"fmt"
	"net/http"

	"github.com/sisoputnfrba/tp-golang/memoria/internal"
)

func main() {
	mux := http.NewServeMux()
	h := internal.NewHandler()

	// Recepción de valores
	mux.HandleFunc("/pedir-acceso", h.RecibirPeticionAcceso)      // Kernel --> Memoria
	mux.HandleFunc("/enviar-instrucciones", h.RecibirInstruccion) // CPU --> Memoria

	// Envío de valores
	mux.HandleFunc("/recibir-puerto", h.EnviarInstrucciones) // Memoria --> CPU

	memoriaAddress := fmt.Sprintf("%s:%d", h.Config.IpMemory, h.Config.PortMemory)
	if err := http.ListenAndServe(memoriaAddress, mux); err != nil {
		h.Log.Error("Error starting server", "err", err)
		panic(err)
	}
}
