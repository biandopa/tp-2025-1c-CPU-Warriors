package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/sisoputnfrba/tp-golang/utils/log"
)

type Interrupcion struct {
	Interrupcion string `json:"interrupcion"`
}

// EnviarInterrupcion envia una Interrupcion al Cpu
func (h *Handler) EnviarInterrupcion(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Creo una interrupción
	interrupcion := Interrupcion{
		Interrupcion: "Interrupcion1",
	}

	// Convierto la estructura del proceso a un []bytes (formato en el que se envían las peticiones)
	body, err := json.Marshal(interrupcion)
	if err != nil {
		h.Log.ErrorContext(ctx, "error codificando mensaje",
			log.ErrAttr(err),
		)
	}

	url := fmt.Sprintf("http://%s:%d/instrucciones", h.Config.IpCPU, h.Config.PortCPU)
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(body))
	if err != nil {
		h.Log.Error("error enviando mensaje",
			log.ErrAttr(err),
			log.StringAttr("ip", h.Config.IpCPU),
			log.IntAttr("puerto", h.Config.PortCPU),
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

	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("ok"))
}
