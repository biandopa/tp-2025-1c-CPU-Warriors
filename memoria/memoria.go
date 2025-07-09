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
	//mux.HandleFunc("POST /kernel/acceso", h.RecibirPeticionAcceso)                   // Kernel --> Memoria
	//mux.HandleFunc("POST /cpu/acceso", h.RecibirPeticionAcceso)                      // CPU --> Memoria (READ/WRITE)
	mux.HandleFunc("POST /cpu/instruccion", h.RecibirInstruccion) // CPU --> Memoria
	mux.HandleFunc("GET /cpu/instruccion", h.EnviarInstruccion)   // Memoria --> CPU
	//mux.HandleFunc("POST /kernel/proceso", h.RecibirProceso)                         // Kernel --> Memoria
	mux.HandleFunc("GET /kernel/espacio-disponible", h.ConsultarEspacioEInicializar)       // Kernel --> Memoria
	mux.HandleFunc("/kernel/cargar-memoria-de-sistema", h.CargarProcesoEnMemoriaDeSistema) // Kernel --> Memoria
	mux.HandleFunc("GET /kernel/swap-proceso", h.PasarProcesoASwap)                        // Kernel --> Memoria
	mux.HandleFunc("/kernel/dump-proceso", h.DumpProceso)                                  // Kernel --> Memoria
	mux.HandleFunc("/kernel/acceso-a-tabla", h.AccesoATabla)
	mux.HandleFunc("POST /kernel/fin-proceso/{pid}", h.FinalizarProceso) // Kernel --> Memoria

	memoriaAddress := fmt.Sprintf("%s:%d", h.Config.IpMemory, h.Config.PortMemory)
	if err := http.ListenAndServe(memoriaAddress, mux); err != nil {
		h.Log.Error("Error starting server", log.ErrAttr(err))
		panic(err)
	}
}
