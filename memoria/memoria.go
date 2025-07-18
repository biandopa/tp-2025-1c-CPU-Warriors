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

	mux.HandleFunc("POST /cpu/instruccion", h.RecibirInstruccion)                          // CPU --> Memoria
	mux.HandleFunc("GET /cpu/instruccion", h.EnviarInstruccion)                            // Memoria --> CPU
	mux.HandleFunc("POST /cpu/escritura", h.EscribirPagina)                                // CPU --> Memoria
	mux.HandleFunc("POST /cpu/lectura", h.LeerPagina)                                      // CPU --> Memoria
	mux.HandleFunc("GET /cpu/pagina-a-frame", h.BuscarMarcoPorPagina)                      // CPU --> Memoria
	mux.HandleFunc("GET /kernel/espacio-disponible", h.ConsultarEspacioEInicializar)       // Kernel --> Memoria
	mux.HandleFunc("/kernel/cargar-memoria-de-sistema", h.CargarProcesoEnMemoriaDeSistema) // Kernel --> Memoria
	mux.HandleFunc("GET /kernel/swap-proceso", h.PasarProcesoASwap)                        // Kernel --> Memoria
	mux.HandleFunc("/kernel/dump-proceso", h.DumpProceso)                                  // Kernel --> Memoria
	mux.HandleFunc("/kernel/acceso-a-tabla", h.AccesoATabla)
	mux.HandleFunc("POST /kernel/fin-proceso", h.FinalizarProceso)                  // Kernel --> Memoria
	mux.HandleFunc("GET /cpu/page-size-y-entries", h.RetornarPageSizeYEntries)      // CPU --> Memoria
	mux.HandleFunc("POST /cpu/actualizar-pag-completa", h.ActualizarPaginaCompleta) // CPU --> Memoria

	memoriaAddress := fmt.Sprintf("%s:%d", h.Config.IpMemory, h.Config.PortMemory)
	if err := http.ListenAndServe(memoriaAddress, mux); err != nil {
		h.Log.Error("Error starting server", log.ErrAttr(err))
		panic(err)
	}
}
