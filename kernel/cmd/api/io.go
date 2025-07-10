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
		if ioDevice.Nombre == ioIdentificacionPeticion.Nombre && !ioDevice.Estado {
			// Liberar el dispositivo IO
			ioIdentificacion[i].Estado = true
			ioIdentificacion[i].ProcesoID = -1 // Limpiar el PID asociado (usar -1 para indicar sin proceso)
			ioIdentificacion[i].Cola = ""      // Limpiar la cola asociada

			h.Log.Debug("Dispositivo IO liberado",
				log.StringAttr("dispositivo", ioDevice.Nombre),
				log.IntAttr("proceso_liberado", ioIdentificacionPeticion.ProcesoID),
			)

			// Procesar la cola de espera para este dispositivo
			ioWaitQueuesMutex.Lock()
			if queue, exists := ioWaitQueues[ioDevice.Nombre]; exists && len(queue) > 0 {
				// Obtener el primer proceso en espera (FIFO)
				nextWaitInfo := queue[0]
				ioWaitQueues[ioDevice.Nombre] = queue[1:] // Remover de la cola

				h.Log.Debug("Procesando siguiente proceso en cola de espera IO",
					log.StringAttr("dispositivo", ioDevice.Nombre),
					log.IntAttr("proceso", nextWaitInfo.PID),
					log.IntAttr("tiempo", nextWaitInfo.TimeSleep),
				)

				// Marcar el dispositivo como ocupado por el siguiente proceso
				ioIdentificacion[i].Estado = false
				ioIdentificacion[i].ProcesoID = nextWaitInfo.PID
				ioIdentificacion[i].Cola = "blocked"

				ioWaitQueuesMutex.Unlock()

				// Enviar petición a IO para el proceso en espera
				go h.Planificador.EnviarUsleep(ioDevice.Puerto, ioDevice.IP, nextWaitInfo.PID, nextWaitInfo.TimeSleep)
			} else {
				ioWaitQueuesMutex.Unlock()
			}

			break
		}
	}

	//Log obligatorio: Fin de IO
	//Fin de IO: "## (<PID>) finalizó IO y pasa a READY"
	h.Log.Info(fmt.Sprintf("## (%d) finalizó IO y pasa a READY", ioIdentificacionPeticion.ProcesoID))

	// Buscar el proceso en cualquier cola (primero en blocked, luego en suspended_blocked)
	proceso := h.Planificador.BuscarProcesoEnCola(ioIdentificacionPeticion.ProcesoID, "blocked")
	if proceso == nil {
		// Si no está en blocked, buscar en suspended_blocked
		proceso = h.Planificador.BuscarProcesoEnCola(ioIdentificacionPeticion.ProcesoID, "suspended_blocked")
	}

	if proceso != nil {
		//Aviso al kernel que el proceso termino su IO para que revise si esta suspendido
		go h.Planificador.ManejarFinIO(proceso)
	} else {
		h.Log.Error("Proceso no encontrado en ninguna cola al finalizar IO",
			log.IntAttr("PID", ioIdentificacionPeticion.ProcesoID),
			log.StringAttr("Cola_original", ioIdentificacionPeticion.Cola),
		)
	}
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("ok"))
}
