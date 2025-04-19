package main

import (
	"fmt"
	"net/http"

	"github.com/sisoputnfrba/tp-golang/memoria/internal"
	"github.com/sisoputnfrba/tp-golang/utils/log"
)

const (
	configFilePath = "./configs/config.json"
)

func main() {
	mux := http.NewServeMux()
	h := internal.NewHandler(configFilePath)

	// Recepción de valores
	mux.HandleFunc("POST /kernel/acceso", h.RecibirPeticionAcceso)  // Kernel --> Memoria
	mux.HandleFunc("POST /cpu/instrucciones", h.RecibirInstruccion) // CPU --> Memoria

	// Envío de valores
	mux.HandleFunc("POST /cpu/instrucciones", h.EnviarInstrucciones) // Memoria --> CPU

	memoriaAddress := fmt.Sprintf("%s:%d", h.Config.IpMemory, h.Config.PortMemory)
	if err := http.ListenAndServe(memoriaAddress, mux); err != nil {
		h.Log.Error("Error starting server", log.ErrAttr(err))
		panic(err)
	}
}
