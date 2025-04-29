package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
)

func (h *Handler) ConexionInicial(archivoNombre, tamanioProceso string) {
	h.Log.Debug("Conexión Inicial",
		slog.Attr{Key: "archivo", Value: slog.StringValue(archivoNombre)},
		slog.Attr{Key: "tamaño", Value: slog.StringValue(tamanioProceso)},
		slog.Attr{Key: "config", Value: slog.AnyValue(h.Config)},
	)

	body, err := json.Marshal(tamanioProceso)
	if err != nil {
		h.Log.Error("Error al serializar tamanioProceso",
			slog.Attr{Key: "error", Value: slog.StringValue(err.Error())},
		)
		return
	}

	url := fmt.Sprintf("http://%s:%d/kernel/acceso", h.Config.IpMemory, h.Config.PortMemory)
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(body))
	if err != nil {
		h.Log.Error("Error enviando mensaje a memoria",
			slog.Attr{Key: "ip", Value: slog.StringValue(h.Config.IpMemory)},
			slog.Attr{Key: "puerto", Value: slog.IntValue(h.Config.PortMemory)},
			slog.Attr{Key: "error", Value: slog.StringValue(err.Error())},
		)
	}

	if resp != nil {
		h.Log.Info("Respuesta del servidor",
			slog.Attr{Key: "status", Value: slog.StringValue(resp.Status)},
			slog.Attr{Key: "body", Value: slog.StringValue(string(body))},
		)
	} else {
		h.Log.Info("Respuesta del servidor: nil")
	}
}

func (h *Handler) ConexionInicialIO(w http.ResponseWriter, r *http.Request) {
	decoder := json.NewDecoder(r.Body)
	err := decoder.Decode(&ioIdentificacion)
	if err != nil {
		h.Log.Error("Error al decodificar ioIdentificacion",
			slog.Attr{Key: "error", Value: slog.StringValue(err.Error())},
		)

		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte("Error al decodificar ioIdentificacion"))
		return
	}

	h.Log.Debug("Me llego la conexion de un IO",
		slog.Attr{Key: "ioIdentificacion", Value: slog.AnyValue(ioIdentificacion)},
	)

	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("ok"))
}

func (h *Handler) ConexionInicialCPU(w http.ResponseWriter, r *http.Request) {
	decoder := json.NewDecoder(r.Body)
	err := decoder.Decode(&identificacionCPU)
	if err != nil {
		h.Log.Error("Error al decodificar ioIdentificacion",
			slog.Attr{Key: "error", Value: slog.StringValue(err.Error())},
		)
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte("Error al decodificar ioIdentificacion"))
	}

	h.Log.Debug("Me llego la conexion de CPU",
		slog.Attr{Key: "identificacionCPU", Value: slog.AnyValue(identificacionCPU)},
	)

	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("ok"))
}

// TODO: usarla donde sea necesario
func (h *Handler) EnviarPeticionAIO(w http.ResponseWriter, tiempoSleep int) {
	body, err := json.Marshal(tiempoSleep)
	if err != nil {
		h.Log.Error("Error codificando tiempoSleep",
			slog.Attr{Key: "error", Value: slog.StringValue(err.Error())},
		)
		http.Error(w, "error codificando mensaje", http.StatusInternalServerError)
		return
	}

	url := fmt.Sprintf("http://%s:%d/io/peticion", ioIdentificacion.IP, ioIdentificacion.Puerto)
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(body))
	if err != nil {
		h.Log.Error("Error enviando mensaje a peticion",
			slog.Attr{Key: "error", Value: slog.StringValue(err.Error())},
		)
		http.Error(w, "error enviando mensaje", http.StatusBadRequest)
		return
	}

	if resp != nil {
		h.Log.Debug("Respuesta del servidor",
			slog.Attr{Key: "status", Value: slog.StringValue(resp.Status)},
			slog.Attr{Key: "body", Value: slog.StringValue(string(body))},
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
			slog.Attr{Key: "error", Value: slog.StringValue(err.Error())},
		)
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte("Error al decodificar ioIdentificacion"))
		return
	}

	//TODO: Buscar en la lista de ioIdentificacion y cambiarle es status
	h.Log.Debug("Me llego la peticion Finalizada de IO",
		slog.Attr{Key: "ioIdentificacion", Value: slog.AnyValue(ioIdentificacionPeticion)},
	)

	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("ok"))
}
