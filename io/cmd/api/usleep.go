package api

import (
	"encoding/json"
	"fmt"
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

	//Log obligatorio: Inicio de IO
	//“## PID: <PID> - Inicio de IO - Tiempo: <TIEMPO_IO>”
	h.Log.Info(fmt.Sprintf("%d PID - Inicio de IO - Tiempo: %d", usleep.PID, usleep.TiempoSleep),
		log.IntAttr("PID", usleep.PID),
		log.IntAttr("Tiempo", usleep.TiempoSleep),
	)

	// Simula el tiempo de espera
	time.Sleep(time.Duration(usleep.TiempoSleep) * time.Millisecond)

	//Log obligatorio: Fin de IO
	//“## PID: <PID> - Fin de IO”.
	h.Log.Info(fmt.Sprintf("%d PID - Fin de IO", usleep.PID),
		log.IntAttr("PID", usleep.PID),
	)

	ioFinOk := finIO{
		PID:         usleep.PID,
		Dispositivo: h.Nombre,
	}
	body, _ := json.Marshal(ioFinOk)

	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(body)
}
