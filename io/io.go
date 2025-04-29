package main

import (
	"fmt"
	"log/slog"
	"net/http"
	"os"

	"github.com/sisoputnfrba/tp-golang/io/cmd/api"
)

const (
	configFilePath = "./configs/config.json"
)

func main() {
	h := api.NewHandler(configFilePath)

	//para que tome el argumento debe ingresarse asi "go run io.go NOMBRE"
	api.NombreIO = os.Args[1]

	h.Log.Debug("Inicializando interfaz IO",
		slog.Attr{Key: "nombre", Value: slog.StringValue(api.NombreIO)},
	)

	//IO --> Kernel  (le enviará su nombre, ip y puerto)  HANDSHAKE
	h.ConeccionInicial()

	mux := http.NewServeMux()

	err := http.ListenAndServe(fmt.Sprintf(":%d", h.Config.PortIo), mux)
	if err != nil {
		panic(err)
	}

	//Kernel --> IO (usleep) LISTO
	mux.HandleFunc("/kernel/usleep", h.EjecutarPeticion)
	//IO --> Kernel  (respuesta de solicitud finalizada) LISTO
}
