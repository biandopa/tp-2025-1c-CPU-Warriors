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

	// Crea la estructura de la peticion
	usleep := Usleep{}
	usleep.PID = pid
	usleep.TiempoSleep = tiempoSleep

	// Serializa la peticion a JSON
	body, err := json.Marshal(usleep)
	if err != nil {
		h.Log.Error("Error al serializar la peticion",
			log.ErrAttr(err),
		)
		return
	}

	// Construye la URL de la peticion
	url := fmt.Sprintf("http://%s:%d/kernel/usleep", io.IP, io.Puerto)

	// Envia la peticion a la IO
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(body))
	if err != nil {
		h.Log.Error("Error enviando mensaje a peticion",
			log.ErrAttr(err),
		)
		return
	}

	// Maneja la respuesta del servidor
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
	//Log obligatorio: Fin de IO
	//Fin de IO: "## (<PID>) finalizó IO y pasa a READY"
	h.Log.Info(fmt.Sprintf("## (%d) finalizó IO y pasa a READY", ioIdentificacionPeticion.ProcesoID))

	proceso := h.Planificador.BuscarProcesoEnCola(ioIdentificacionPeticion.ProcesoID, ioIdentificacionPeticion.Cola)
	//Aviso al kernel que el proceso termino su IO para que revise si esta suspendido
	go h.Planificador.ManejarFinIO(proceso)
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("ok"))
}
