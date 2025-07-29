package main

import (
	"fmt"
	"net/http"
	"os"

	"github.com/sisoputnfrba/tp-golang/memoria/cmd/api"
	"github.com/sisoputnfrba/tp-golang/utils/log"
)

const (
	configFilePath = "./configs/"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Error: Missing required argument 'CONFIG_ID'. Usage: go run memoria.go {{CONFIG_ID}}")
		os.Exit(1)
	}
	configID := os.Args[1]
	configFile := configFilePath + configID + ".json"

	mux := http.NewServeMux()
	h := api.NewHandler(configFile)

	mux.HandleFunc("POST /cpu/instruccion", h.RecibirInstruccion)                          // CPU --> Memoria
	mux.HandleFunc("POST /cpu/escritura", h.EscribirPagina)                                // CPU --> Memoria
	mux.HandleFunc("POST /cpu/lectura", h.LeerPagina)                                      // CPU --> Memoria
	mux.HandleFunc("POST /cpu/lectura-completa", h.LeerPaginaCompleta)                     // CPU --> Memoria
	mux.HandleFunc("GET /cpu/pagina-a-frame", h.BuscarMarcoPorPagina)                      // CPU --> Memoria
	mux.HandleFunc("GET /kernel/espacio-disponible", h.ConsultarEspacioEInicializar)       // Kernel --> Memoria
	mux.HandleFunc("/kernel/cargar-memoria-de-sistema", h.CargarProcesoEnMemoriaDeSistema) // Kernel --> Memoria
	mux.HandleFunc("GET /kernel/swap-proceso", h.PasarProcesoASwap)                        // Kernel --> Memoria
	mux.HandleFunc("/kernel/dump-proceso", h.DumpProceso)                                  // Kernel --> Memoria
	mux.HandleFunc("POST /kernel/fin-proceso", h.FinalizarProceso)                         // Kernel --> Memoria
	mux.HandleFunc("GET /cpu/page-size-y-entries", h.RetornarPageSizeYEntries)             // CPU --> Memoria
	mux.HandleFunc("POST /cpu/actualizar-pag-completa", h.ActualizarPaginaCompleta)        // CPU --> Memoria

	memoriaAddress := fmt.Sprintf("%s:%d", h.Config.IpMemory, h.Config.PortMemory)
	if err := http.ListenAndServe(memoriaAddress, mux); err != nil {
		h.Log.Error("Error starting server", log.ErrAttr(err))
		panic(err)
	}
}
