package api

import (
	"bytes"
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
	//"## PID: <PID> - Inicio de IO - Tiempo: <TIEMPO_IO>"
	h.Log.Info(fmt.Sprintf("## PID: %d - Inicio de IO - Tiempo: %d", usleep.PID, usleep.TiempoSleep),
		log.IntAttr("PID", usleep.PID),
		log.IntAttr("Tiempo", usleep.TiempoSleep),
	)

	// Simula el tiempo de espera
	time.Sleep(time.Duration(usleep.TiempoSleep) * time.Millisecond)

	//Log obligatorio: Fin de IO
	//"## PID: <PID> - Fin de IO".
	h.Log.Info(fmt.Sprintf("## PID: %d - Fin de IO", usleep.PID),
		log.IntAttr("PID", usleep.PID),
	)

	// Notificar al kernel que el proceso terminó el IO
	err = h.notificarKernelFinIO(usleep.PID)
	if err != nil {
		h.Log.Error("Error al notificar kernel fin de IO",
			log.ErrAttr(err),
			log.IntAttr("PID", usleep.PID),
		)
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte("Error al notificar kernel"))
		return
	}

	// Respuesta exitosa
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("IO operation completed successfully"))
}

// notificarKernelFinIO envía una notificación POST al kernel cuando termina una operación IO
func (h *Handler) notificarKernelFinIO(pid int) error {
	// Estructura para enviar al kernel (compatible con lo que espera el endpoint /io/peticion-finalizada)
	finIOData := IOIdentificacion{
		Nombre:    h.Nombre,
		IP:        h.Config.IpIo,
		Puerto:    h.Config.PortIo,
		ProcesoID: pid,
		Cola:      "blocked", // El proceso estaba en la cola de blocked durante el IO
	}

	// Serializar la estructura a JSON
	body, err := json.Marshal(finIOData)
	if err != nil {
		return fmt.Errorf("error serializing fin IO data: %w", err)
	}

	// Enviar la solicitud POST al kernel
	url := fmt.Sprintf("http://%s:%d/io/peticion-finalizada", h.Config.IpKernel, h.Config.PortKernel)
	resp, err := h.HttpClient.Post(url, "application/json", bytes.NewBuffer(body))
	if err != nil {
		return fmt.Errorf("error sending POST to kernel: %w", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	// Verificar que la respuesta sea exitosa
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("kernel returned non-OK status: %s", resp.Status)
	}

	h.Log.Debug("Kernel notificado exitosamente de fin de IO",
		log.IntAttr("PID", pid),
		log.StringAttr("dispositivo", h.Nombre),
		log.StringAttr("kernel_response", resp.Status),
	)

	return nil
}
