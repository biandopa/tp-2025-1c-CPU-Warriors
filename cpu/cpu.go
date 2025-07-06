package main

import (
	"fmt"
	"net/http"
	"os"

	"github.com/sisoputnfrba/tp-golang/cpu/cmd/api"
	"github.com/sisoputnfrba/tp-golang/utils/log"
)

const (
	configFilePath = "./configs/"
)

func main() {
	mux := http.NewServeMux()

	//para que tome el argumento debe ingresarse asi "go run cpu.go Identificador"
	if len(os.Args) < 2 {
		fmt.Println("Error: Missing required argument 'Identificador'. Usage: go run cpu.go {{CPU_ID}}")
		os.Exit(1)
	}

	identificadorCPU := os.Args[1]

	configFile := configFilePath + identificadorCPU + ".json"

	h := api.NewHandler(configFile)

	h.Log.Debug("Inicializando interfaz CPU",
		log.StringAttr("id", identificadorCPU),
	)

	//IO --> Kernel  (le enviará su nombre, ip y puerto)  HANDSHAKE
	h.EnviarIdentificacion(identificadorCPU)

	// Recepción de valores
	mux.HandleFunc("POST /memoria/instrucciones", h.RecibirInstrucciones)  // Memoria --> CPU
	mux.HandleFunc("POST /kernel/procesos", h.RecibirProcesos)             // Kernel --> CPU
	mux.HandleFunc("POST /kernel/interrupciones", h.RecibirInterrupciones) // Kernel --> CPU

	// Nota: Le pasamos por argumento el puerto para que levante muchas CPUs
	cpuAddress := fmt.Sprintf("%s:%d", h.Config.IpCpu, h.Config.PortCpu)
	if err := http.ListenAndServe(cpuAddress, mux); err != nil {
		h.Log.Error("Error starting server", log.ErrAttr(err))
		panic(err)
	}
}
