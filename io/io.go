package main

import (
	"fmt"
	"net/http"
	"os"

	"github.com/sisoputnfrba/tp-golang/io/internal"
)

func main() {

	//para que tome el argumento debe ingresarse asi "go run io.go NOMBRE"
	internal.NombreIO = os.Args[1]

	internal.ClientConfig = internal.IniciarConfiguracion("config.json")

	//BORRAR
	fmt.Println("Inicializando interfaz IO con nombre:", internal.NombreIO)

	//IO --> Kernel  (le enviará su nombre, ip y puerto)  HANDSHAKE
	internal.ConeccionInicial()

	mux := http.NewServeMux()

	err := http.ListenAndServe(fmt.Sprintf(":%d", internal.ClientConfig.PortIo), mux)
	if err != nil {
		panic(err)
	}

	//Kernel --> IO (usleep) LISTO
	mux.HandleFunc("/petiocionKernel", internal.EjecutarPeticion)
	//IO --> Kernel  (respuesta de solicitud finalizada) LISTO
}
