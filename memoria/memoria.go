package main

import (
	"fmt"
	"net/http"

	"github.com/sisoputnfrba/tp-golang/memoria/cmd/api"
	"github.com/sisoputnfrba/tp-golang/utils/log"
)

const (
	configFilePath = "./configs/config.json"
)

func main() {
	mux := http.NewServeMux()
	h := api.NewHandler(configFilePath)

	// Recepción de valores
	mux.HandleFunc("POST /kernel/acceso", h.RecibirPeticionAcceso)                 // Kernel --> Memoria
	mux.HandleFunc("POST /cpu/instruccion", h.RecibirInstruccion)                  // CPU --> Memoria
	mux.HandleFunc("POST /kernel/proceso", h.RecibirProceso)                       // Kernel --> Memoria
	mux.HandleFunc("GET /kernel/espacio-disponible", h.ConsultarEspacioDisponible) // Kernel --> Memoria
	mux.HandleFunc("POST /kernel/fin-proceso/{pid}", h.FinalizarProceso)           // Kernel --> Memoria
	mux.HandleFunc("GET /kernel/archivo-instrucciones", h.RecibirInstrucciones)    // Kernel --> Memoria

	// Envío de valores
	mux.HandleFunc("POST /cpu/instrucciones", h.EnviarInstrucciones) // Memoria --> CPU

	memoriaAddress := fmt.Sprintf("%s:%d", h.Config.IpMemory, h.Config.PortMemory)
	if err := http.ListenAndServe(memoriaAddress, mux); err != nil {
		h.Log.Error("Error starting server", log.ErrAttr(err))
		panic(err)
	}
}
