package main

import (
	"net/http"

	"github.com/sisoputnfrba/tp-golang/cpu/internal"
	"github.com/sisoputnfrba/tp-golang/utils/log"
)

func main() {
	logger := log.BuildLogger()

	handler := http.NewServeMux()

	handler.HandleFunc("/instrucciones", internal.RecibirInstrucciones)
	handler.HandleFunc("/procesos", internal.RecibirProcesos)
	handler.HandleFunc("/interrupciones", internal.RecibirInterrupciones)

	if err := http.ListenAndServe(":8080", handler); err != nil {
		logger.Error("Error starting server", "err", err)
		panic(err)
	}
}
