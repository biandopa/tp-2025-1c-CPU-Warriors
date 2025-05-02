package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/sisoputnfrba/tp-golang/utils/log"
)

func (h *Handler) ConexionInicialMemoria(archivoNombre, tamanioProceso string) {
	h.Log.Debug("Conexión Inicial",
		log.StringAttr("archivo", archivoNombre),
		log.StringAttr("tamaño", tamanioProceso),
		log.AnyAttr("config", h.Config),
	)

	body, err := json.Marshal(tamanioProceso)
	if err != nil {
		h.Log.Error("Error al serializar tamanioProceso",
			log.ErrAttr(err),
		)
		return
	}

	url := fmt.Sprintf("http://%s:%d/kernel/acceso", h.Config.IpMemory, h.Config.PortMemory)
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(body))
	if err != nil {
		h.Log.Error("Error enviando mensaje a memoria",
			log.ErrAttr(err),
			log.StringAttr("ip", h.Config.IpMemory),
			log.IntAttr("puerto", h.Config.PortMemory),
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

func (h *Handler) ConexionInicialIO(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	ioInfo := IOIdentificacion{}

	// Leer el cuerpo de la solicitud
	decoder := json.NewDecoder(r.Body)
	err := decoder.Decode(&ioInfo)
	if err != nil {
		h.Log.ErrorContext(ctx, "Error al decodificar ioIdentificacion",
			log.ErrAttr(err),
		)

		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte("Error al decodificar ioIdentificacion"))
		return
	}

	h.Config.IpIo = ioInfo.IP
	h.Config.PortIo = ioInfo.Puerto
	// Agregar nombre si es necesario

	h.Log.DebugContext(ctx, "Me llego la conexion de un IO",
		log.StringAttr("nombre", ioInfo.Nombre),
	)

	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("ok"))
}

// TRANSFORMAR ESTO A UNA LISTA DE CPUS
func (h *Handler) ConexionInicialCPU(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	identificacionCPU := CPUIdentificacion{}

	h.Log.DebugContext(ctx, "Me llego la conexion de un CPU??")

	// Leer el cuerpo de la solicitud
	decoder := json.NewDecoder(r.Body)

	// Decodificar el cuerpo de la solicitud en la estructura identificacionCPU
	err := decoder.Decode(&identificacionCPU)
	if err != nil {
		h.Log.ErrorContext(ctx, "Error al decodificar ioIdentificacion",
			log.ErrAttr(err),
		)
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte("Error al decodificar ioIdentificacion"))
	}

	h.Log.DebugContext(ctx, "Me llego la conexion de CPU",
		log.AnyAttr("identificacionCPU", identificacionCPU),
	)

	identificacionCPU.ESTADO = true

	h.CPUConectadas = append(h.CPUConectadas, identificacionCPU)

	h.Log.DebugContext(ctx, "Lista actual de CPUs conectadas",
		log.AnyAttr("CPUConectadas", h.CPUConectadas),
	)

	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("ok"))
}
