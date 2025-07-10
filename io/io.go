package main

import (
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"

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

	/*
		Configurar manejo de señales para finalización controlada
		Cómo funciona:
		  # 1. Iniciar módulo IO
		    go run io.go TECLADO

		  # 2. En otra terminal,
		  Enviar señal SIGINT
		    # Ctrl+C en la terminal del IO
		  O enviar señal SIGTERM
		    # kill -TERM <PID_DEL_PROCESO_IO>

		# El módulo IO:
		# - Detectará la señal
		# - Notificará al kernel su desconexión
		# - Finalizará de manera controlada
		# - Mostrará logs informativos
	*/
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	// Goroutine para manejar las señales
	go func() {
		sig := <-sigs
		h.Log.Debug("Señal recibida, finalizando módulo IO de manera controlada",
			log.StringAttr("signal", sig.String()),
			log.StringAttr("nombreIO", nombreIO),
		)

		// Notificar al kernel la desconexión
		err := h.NotificarDesconexionKernel(nombreIO)
		if err != nil {
			h.Log.Error("Error al notificar desconexión al kernel",
				log.ErrAttr(err),
			)
		} else {
			h.Log.Debug("Kernel notificado de la desconexión exitosamente")
		}

		// Finalizar el programa
		os.Exit(0)
	}()

	//IO --> Kernel  (le enviará su nombre, ip y puerto)  HANDSHAKE
	h.ConexionInicialKernel(nombreIO)

	mux := http.NewServeMux()

	mux.HandleFunc("/kernel/usleep", h.EjecutarPeticion)

	err := http.ListenAndServe(fmt.Sprintf(":%d", h.Config.PortIo), mux)
	if err != nil {
		panic(err)
	}
}
