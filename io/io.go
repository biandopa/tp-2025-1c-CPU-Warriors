package main

import (
	"fmt"
	"net/http"
	"os"

	"github.com/sisoputnfrba/tp-golang/io/internal"
)

func main() {

	//para que tome el argumento debe ingresarse asi "go run io.go NOMBRE"
	interfazIo := os.Args[1]

	internal.ClientConfig = internal.IniciarConfiguracion("config.json")

	//BORRAR
	fmt.Println("Inicializando interfaz IO con nombre:", interfazIo)

	//IO --> Kernel  (le enviará su nombre, ip y puerto)  HANDSHAKE
	internal.ConeccionInicial(interfazIo, internal.ClientConfig)

	mux := http.NewServeMux()

	err := http.ListenAndServe(fmt.Sprintf(":%d", internal.ClientConfig.PortIo), mux)
	if err != nil {
		panic(err)
	}

	//Kernel --> IO (usleep)
	//mux.HandleFunc("/petiocionKernel", internal.EjecutarPeticion)
	//IO --> Kernel  (respuesta de solicitud finalizada) posiblemnte va dentro del EjecutarPeticion
}
