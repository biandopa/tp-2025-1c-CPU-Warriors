package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/sisoputnfrba/tp-golang/utils/log"
	"net/http"
)

func (h *Handler) EnviarIdentificacion(nombre string) {
	data := map[string]interface{}{
		"ip":     h.Config.IpCpu,
		"puerto": h.Config.PortCpu,
		"id":     nombre,
	}

	body, err := json.Marshal(data)
	if err != nil {
		h.Log.Error("Error al serializar ioIdentificacion",
			log.ErrAttr(err),
		)
		return
	}

	url := fmt.Sprintf("http://%s:%d/cpu/conexion-inicial", h.Config.IpKernel, h.Config.PortKernel)
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(body))
	if err != nil {
		h.Log.Error("error enviando mensaje",
			log.ErrAttr(err),
			log.StringAttr("ip", h.Config.IpCpu),
			log.IntAttr("puerto", h.Config.PortCpu),
		)
	}

	if resp != nil {
		h.Log.Info("Respuesta del servidor",
			log.StringAttr("status", resp.Status),
			log.StringAttr("body", string(body)),
		)
	} else {
		h.Log.Info("Respuesta del servidor: nil")
	}
}
