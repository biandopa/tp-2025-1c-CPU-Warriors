package internal

import (
	"encoding/json"
	"log"
	"net/http"
)

type IOIdentificacion struct {
	Nombre string `json:"nombre"`
	IP     string `json:"ip"`
	Puerto int    `json:"puerto"`
}

func ConeccionInicialIO(w http.ResponseWriter, r *http.Request) {
	decoder := json.NewDecoder(r.Body)
	var ioIdentificacion IOIdentificacion
	err := decoder.Decode(&ioIdentificacion)
	if err != nil {
		log.Printf("Error al decodificar mensaje: %s\n", err.Error())
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("Error al decodificar mensaje"))
		return
	}

	log.Println("Me llego un mensaje de un cliente")
	log.Printf("%+v\n", ioIdentificacion)

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("ok"))
}
