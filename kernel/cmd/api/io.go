package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/sisoputnfrba/tp-golang/utils/log"
)

type Usleep struct {
	PID         int `json:"pid"`
	TiempoSleep int `json:"tiempo_sleep"`
}

// EnviarPeticionAIO Envia la peticion de usar la IO
func (h *Handler) EnviarPeticionAIO(tiempoSleep int, io IOIdentificacion, pid int) {

	usleep := Usleep{}
	usleep.PID = 123
	usleep.TiempoSleep = tiempoSleep

	body, err := json.Marshal(usleep)
	if err != nil {
		h.Log.Error("Error al serializar la peticion",
			log.ErrAttr(err),
		)
		return
	}

	url := fmt.Sprintf("http://%s:%d/kernel/usleep", io.IP, io.Puerto)
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(body))
	if err != nil {
		h.Log.Error("Error enviando mensaje a peticion",
			log.ErrAttr(err),
		)
		return
	}

	if resp != nil {
		h.Log.Debug("Respuesta del servidor",
			log.StringAttr("status", resp.Status),
			log.StringAttr("body", string(body)),
		)
	} else {
		h.Log.Debug("Respuesta del servidor: nil")
	}
}

// TerminoPeticionIO Devuelve la peticion luego de usar la IO
func (h *Handler) TerminoPeticionIO(w http.ResponseWriter, r *http.Request) {
	decoder := json.NewDecoder(r.Body)
	var ioIdentificacionPeticion IOIdentificacion
	err := decoder.Decode(&ioIdentificacionPeticion)
	if err != nil {
		h.Log.Error("Error al decodificar ioIdentificacion",
			log.ErrAttr(err),
		)
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte("Error al decodificar ioIdentificacion"))
		return
	}

	//TODO: Buscar en la lista de ioIdentificacion y cambiarle es status
	h.Log.Debug("Me llego la peticion Finalizada de IO",
		log.AnyAttr("ioIdentificacionPeticion", ioIdentificacionPeticion),
	)

	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("ok"))
}
