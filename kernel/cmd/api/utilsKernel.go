package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/sisoputnfrba/tp-golang/utils/log"
)

// TODO: usarla donde sea necesario  -> Borrar porque no es una llamada en el handler, sino en internal.
func (h *Handler) EnviarPeticionAIO(w http.ResponseWriter, tiempoSleep int) {
	body, err := json.Marshal(tiempoSleep)
	if err != nil {
		h.Log.Error("Error codificando tiempoSleep",
			log.ErrAttr(err),
		)
		http.Error(w, "error codificando mensaje", http.StatusInternalServerError)
		return
	}

	url := fmt.Sprintf("http://%s:%d/io/peticion", ioIdentificacion.IP, ioIdentificacion.Puerto)
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(body))
	if err != nil {
		h.Log.Error("Error enviando mensaje a peticion",
			log.ErrAttr(err),
		)
		http.Error(w, "error enviando mensaje", http.StatusBadRequest)
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
