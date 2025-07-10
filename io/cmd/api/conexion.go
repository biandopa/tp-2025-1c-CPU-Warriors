package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/sisoputnfrba/tp-golang/utils/log"
)

func (h *Handler) ConexionInicialKernel(nombre string) {
	// Estructura para enviar la identificación del IO al kernel
	data := IOIdentificacion{
		Nombre: nombre,
		IP:     h.Config.IpIo,
		Puerto: h.Config.PortIo,
	}

	// Serializar la estructura a JSON
	body, err := json.Marshal(data)
	if err != nil {
		h.Log.Error("Error al serializar ioIdentificacion",
			log.ErrAttr(err),
		)
		return
	}

	// Enviar la solicitud POST al kernel
	url := fmt.Sprintf("http://%s:%d/io/conexion-inicial", h.Config.IpKernel, h.Config.PortKernel)
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(body))
	if err != nil {
		h.Log.Error("error enviando mensaje",
			log.ErrAttr(err),
			log.StringAttr("ip", h.Config.IpKernel),
			log.IntAttr("puerto", h.Config.PortKernel),
		)
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

// NotificarDesconexionKernel notifica al kernel que el módulo IO se va a desconectar
func (h *Handler) NotificarDesconexionKernel(nombre string) error {
	// Estructura para enviar la notificación de desconexión al kernel
	data := IOIdentificacion{
		Nombre: nombre,
		IP:     h.Config.IpIo,
		Puerto: h.Config.PortIo,
	}

	// Serializar la estructura a JSON
	body, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("error al serializar ioIdentificacion: %w", err)
	}

	// Enviar la solicitud POST al kernel para notificar la desconexión
	url := fmt.Sprintf("http://%s:%d/io/desconexion", h.Config.IpKernel, h.Config.PortKernel)
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(body))
	if err != nil {
		return fmt.Errorf("error enviando notificación de desconexión: %w", err)
	}

	if resp != nil {
		defer func() {
			_ = resp.Body.Close()
		}()

		if resp.StatusCode != http.StatusOK {
			return fmt.Errorf("kernel respondió con status: %s", resp.Status)
		}

		h.Log.Debug("Notificación de desconexión enviada al kernel",
			log.StringAttr("status", resp.Status),
			log.StringAttr("nombre", nombre),
		)
	}

	return nil
}
