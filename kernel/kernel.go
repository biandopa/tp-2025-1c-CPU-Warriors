package main

import (
	"net/http"

	"github.com/sisoputnfrba/tp-golang/kernel/internal"
)

func main() {
	mux := http.NewServeMux()

	mux.HandleFunc("/ioConeccionInicial", internal.ConeccionInicialIO)

	err := http.ListenAndServe(":8001", mux)
	if err != nil {
		panic(err)
	}
}
