package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/sisoputnfrba/tp-golang/kernel/internal/planificadores"
	"github.com/sisoputnfrba/tp-golang/utils/log"
)

// ConexionInicialMemoria Recibe la conexion de la memoria (es unica)
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
		h.Log.Debug("Respuesta del servidor",
			log.StringAttr("status", resp.Status),
			log.StringAttr("body", string(body)),
		)
	} else {
		h.Log.Debug("Respuesta del servidor: nil")
	}
}

// ConexionInicialIO Recibe la lista de IOs
func (h *Handler) ConexionInicialIO(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	var ioInfo IOIdentificacion

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

	ioInfo.Estado = true
	ioInfo.ProcesoID = -1 // Inicializar sin proceso asignado

	ioIdentificacionMutex.Lock()
	ioIdentificacion = append(ioIdentificacion, ioInfo)
	ioIdentificacionMutex.Unlock()

	ioIdentificacionMutex.RLock()
	h.Log.DebugContext(ctx, "Lista de IOs conectadas",
		log.AnyAttr("IOsConectadas", ioIdentificacion),
	)
	ioIdentificacionMutex.RUnlock()

	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("ok"))
}

// DesconexionIO maneja la desconexión de un dispositivo IO
func (h *Handler) DesconexionIO(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	var ioInfo IOIdentificacion

	// Leer el cuerpo de la solicitud
	decoder := json.NewDecoder(r.Body)
	err := decoder.Decode(&ioInfo)
	if err != nil {
		h.Log.ErrorContext(ctx, "Error al decodificar ioIdentificacion para desconexión",
			log.ErrAttr(err),
		)
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte("Error al decodificar ioIdentificacion"))
		return
	}

	h.Log.DebugContext(ctx, "Desconexión de dispositivo IO",
		log.StringAttr("dispositivo", ioInfo.Nombre),
		log.StringAttr("ip", ioInfo.IP),
		log.IntAttr("puerto", ioInfo.Puerto),
	)

	// Encontrar y remover el dispositivo de la lista
	var dispositivoEncontrado *IOIdentificacion
	ioIdentificacionMutex.Lock()
	for i, device := range ioIdentificacion {
		if device.Nombre == ioInfo.Nombre && device.IP == ioInfo.IP && device.Puerto == ioInfo.Puerto {
			dispositivoEncontrado = &device
			// Remover el dispositivo de la lista
			ioIdentificacion = append(ioIdentificacion[:i], ioIdentificacion[i+1:]...)
			break
		}
	}
	ioIdentificacionMutex.Unlock()

	if dispositivoEncontrado != nil {
		// Si había un proceso usando este dispositivo, enviarlo a EXIT
		if dispositivoEncontrado.ProcesoID >= 0 {
			h.Log.Debug(fmt.Sprintf("## (%d) - Proceso enviado a EXIT por desconexión de IO: %s",
				dispositivoEncontrado.ProcesoID, dispositivoEncontrado.Nombre))
			go h.Planificador.FinalizarProcesoEnCualquierCola(dispositivoEncontrado.ProcesoID)
		}
	}

	ioIdentificacionMutex.RLock()
	h.Log.DebugContext(ctx, "Estado actual de IOs conectadas después de desconexión",
		log.AnyAttr("IOsConectadas", ioIdentificacion),
	)
	ioIdentificacionMutex.RUnlock()

	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("ok"))
}

// ConexionInicialCPU Recibe la lista de IOs
func (h *Handler) ConexionInicialCPU(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	identificacionCPU := &planificadores.CpuIdentificacion{}

	// Leer el cuerpo de la solicitud
	decoder := json.NewDecoder(r.Body)

	// Decodificar el cuerpo de la solicitud en la estructura identificacionCPU
	err := decoder.Decode(&identificacionCPU)
	if err != nil {
		h.Log.ErrorContext(ctx, "Error al decodificar cpuIdentificacion",
			log.ErrAttr(err),
		)
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte("Error al decodificar cpuIdentificacion"))
	}

	h.Log.DebugContext(ctx, "Me llego la conexion de CPU",
		log.AnyAttr("identificacionCPU", identificacionCPU),
	)

	identificacionCPU.Estado = true

	h.Planificador.AddCpuConectada(identificacionCPU)

	h.Log.DebugContext(ctx, "Lista actual de CPUs conectadas",
		log.AnyAttr("CPUsConectadas", h.Planificador.CPUsConectadas),
	)

	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("ok"))
}
