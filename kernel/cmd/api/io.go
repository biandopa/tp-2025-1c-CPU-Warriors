package api

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/sisoputnfrba/tp-golang/utils/log"
)

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

	// Buscar el dispositivo IO y marcarlo como libre
	for i, ioDevice := range ioIdentificacion {
		if ioDevice.Nombre == ioIdentificacionPeticion.Nombre && ioDevice.Estado == false {
			// Liberar el dispositivo IO
			ioIdentificacion[i].Estado = true
			ioIdentificacion[i].ProcesoID = 0 // Limpiar el PID asociado
			ioIdentificacion[i].Cola = ""     // Limpiar la cola asociada

			h.Log.Debug("Dispositivo IO liberado",
				log.StringAttr("dispositivo", ioDevice.Nombre),
				log.IntAttr("proceso_liberado", ioIdentificacionPeticion.ProcesoID),
			)

			// Procesar la cola de espera para este dispositivo
			if queue, exists := ioWaitQueues[ioDevice.Nombre]; exists && len(queue) > 0 {
				// Obtener el primer proceso en espera (FIFO)
				nextPID := queue[0]
				ioWaitQueues[ioDevice.Nombre] = queue[1:] // Remover de la cola

				h.Log.Debug("Procesando siguiente proceso en cola de espera IO",
					log.StringAttr("dispositivo", ioDevice.Nombre),
					log.IntAttr("proceso", nextPID),
				)

				// Marcar el dispositivo como ocupado por el siguiente proceso
				ioIdentificacion[i].Estado = false
				ioIdentificacion[i].ProcesoID = nextPID
				ioIdentificacion[i].Cola = "blocked"

				// Buscar el proceso y enviarlo a IO
				// Nota: Necesitaríamos almacenar el tiempo de IO también en la cola de espera
				// Por simplicidad, vamos a usar un tiempo por defecto o implementar después
				h.Log.Info("TODO: Enviar proceso en espera a IO - implementar tiempo de IO guardado")
			}

			break
		}
	}

	//Log obligatorio: Fin de IO
	//Fin de IO: "## (<PID>) finalizó IO y pasa a READY"
	h.Log.Info(fmt.Sprintf("## (%d) finalizó IO y pasa a READY", ioIdentificacionPeticion.ProcesoID))

	proceso := h.Planificador.BuscarProcesoEnCola(ioIdentificacionPeticion.ProcesoID, ioIdentificacionPeticion.Cola)
	//Aviso al kernel que el proceso termino su IO para que revise si esta suspendido
	go h.Planificador.ManejarFinIO(proceso)
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("ok"))
}
