package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"os"
	"time"
)

type IOIdentificacion struct {
	Nombre string `json:"nombre"`
	IP     string `json:"ip"`
	Puerto int    `json:"puerto"`
}

type Config struct {
	IpKernel   string `json:"ip_kernel"`
	PortKernel int    `json:"port_kernel"`
	PortIo     int    `json:"port_io"`
	IpIo       string `json:"ip_io"`
	LogLevel   string `json:"log_level"`
}

type Usleep struct {
	PID         int `json:"pid"`
	TiempoSleep int `json:"tiempo_sleep"`
}

var ClientConfig *Config
var NombreIO string

func IniciarConfiguracion(filePath string) *Config {
	var config *Config
	configFile, err := os.Open(filePath)
	if err != nil {
		log.Fatal(err.Error())
	}
	defer func() {
		_ = configFile.Close()
	}()

	jsonParser := json.NewDecoder(configFile)
	if err = jsonParser.Decode(&config); err != nil {
		log.Fatal(err.Error())
	}

	return config
}

func (h *Handler) ConexionInicial(nombre string) {
	data := IOIdentificacion{
		Nombre: nombre,
		IP:     h.Config.IpIo,
		Puerto: h.Config.PortIo,
	}

	body, err := json.Marshal(data)
	if err != nil {
		h.Log.Error("Error al serializar ioIdentificacion",
			slog.Attr{Key: "error", Value: slog.StringValue(err.Error())},
		)
		return
	}

	url := fmt.Sprintf("http://%s:%d/io/conexion-inicial", h.Config.IpKernel, h.Config.PortKernel)
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(body))
	if err != nil {
		h.Log.Error("error enviando mensaje",
			slog.Attr{Key: "error", Value: slog.StringValue(err.Error())},
			slog.Attr{Key: "ip", Value: slog.StringValue(h.Config.IpKernel)},
			slog.Attr{Key: "puerto", Value: slog.IntValue(h.Config.PortKernel)},
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

func (h *Handler) EjecutarPeticion(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	usleep := Usleep{}

	decoder := json.NewDecoder(r.Body)
	err := decoder.Decode(&usleep)
	if err != nil {
		h.Log.ErrorContext(ctx, "Error al decodificar ioIdentificacion",
			slog.Attr{Key: "error", Value: slog.StringValue(err.Error())},
		)
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte("Error al decodificar ioIdentificacion"))
		return
	}

	h.Log.InfoContext(ctx, "Inicio de IO",
		slog.Attr{Key: "PID", Value: slog.IntValue(usleep.PID)},
		slog.Attr{Key: "Tiempo", Value: slog.IntValue(usleep.TiempoSleep)},
	)

	// Simula el tiempo de espera
	time.Sleep(time.Duration(usleep.TiempoSleep) * time.Millisecond)

	h.Log.InfoContext(ctx, "Fin de IO",
		slog.Attr{Key: "PID", Value: slog.IntValue(usleep.PID)},
	)

	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("ok"))
}
