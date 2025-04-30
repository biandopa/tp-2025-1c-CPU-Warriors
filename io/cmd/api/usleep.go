package api

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/sisoputnfrba/tp-golang/utils/log"
)

func (h *Handler) EjecutarPeticion(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	usleep := Usleep{}

	decoder := json.NewDecoder(r.Body)
	err := decoder.Decode(&usleep)
	if err != nil {
		h.Log.ErrorContext(ctx, "Error al decodificar ioIdentificacion",
			log.ErrAttr(err),
		)
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte("Error al decodificar ioIdentificacion"))
		return
	}

	h.Log.InfoContext(ctx, "Inicio de IO",
		log.IntAttr("PID", usleep.PID),
		log.IntAttr("Tiempo", usleep.TiempoSleep),
	)

	// Simula el tiempo de espera
	time.Sleep(time.Duration(usleep.TiempoSleep) * time.Millisecond)

	h.Log.InfoContext(ctx, "Fin de IO",
		log.IntAttr("PID", usleep.PID),
	)

	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("ok"))
}
