package main

import (
	"fmt"
	"log/slog"
	"net/http"
	"os"

	"github.com/sisoputnfrba/tp-golang/io/cmd/api"
	"github.com/sisoputnfrba/tp-golang/utils/log"
)

const (
	configFilePath = "./configs/config.json"
)

func main() {
	//para que tome el argumento debe ingresarse asi "go run io.go NOMBRE"
	nombreIO := os.Args[1]
	if nombreIO == "" {
		slog.Error("El nombre del IO no puede estar vacío")
		os.Exit(1)
	}

	h := api.NewHandler(configFilePath, nombreIO)

	h.Log.Debug("Inicializando interfaz IO",
		log.StringAttr("nombreIO", nombreIO),
	)

	//IO --> Kernel  (le enviará su nombre, ip y puerto)  HANDSHAKE
	h.ConexionInicialKernel(nombreIO)

	mux := http.NewServeMux()

	mux.HandleFunc("/kernel/usleep", h.EjecutarPeticion)

	err := http.ListenAndServe(fmt.Sprintf(":%d", h.Config.PortIo), mux)
	if err != nil {
		panic(err)
	}
}
