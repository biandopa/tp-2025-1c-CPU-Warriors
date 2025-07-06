package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/sisoputnfrba/tp-golang/utils/log"
)

// PeticionAcceso representa una petición de acceso a memoria desde CPU
type PeticionAcceso struct {
	PID       int    `json:"pid"`
	Direccion string `json:"direccion"`
	Datos     string `json:"datos,omitempty"`   // Solo para WRITE
	Tamanio   int    `json:"tamanio,omitempty"` // Solo para READ
	Operacion string `json:"operacion"`         // "READ" o "WRITE"
}

// RespuestaAcceso representa la respuesta de memoria al CPU
type RespuestaAcceso struct {
	Datos   string `json:"datos,omitempty"` // Solo para READ
	Exito   bool   `json:"exito"`
	Mensaje string `json:"mensaje,omitempty"`
}

func (h *Handler) RecibirPeticionAcceso(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Decode the request body
	var peticion PeticionAcceso
	err := json.NewDecoder(r.Body).Decode(&peticion)
	if err != nil {
		h.Log.ErrorContext(ctx, "Error al decodificar petición de acceso", log.ErrAttr(err))
		http.Error(w, "Petición inválida", http.StatusBadRequest)
		return
	}

	h.Log.DebugContext(ctx, "Petición de acceso recibida",
		log.IntAttr("pid", peticion.PID),
		log.StringAttr("operacion", peticion.Operacion),
		log.StringAttr("direccion", peticion.Direccion),
	)

	// Simular delay de memoria
	if h.Config.MemoryDelay > 0 {
		time.Sleep(time.Duration(h.Config.MemoryDelay) * time.Millisecond)
	}

	var respuesta RespuestaAcceso

	switch peticion.Operacion {
	case "READ":
		// Simulación de lectura de memoria - devolvemos un valor mockeado
		datosMock := fmt.Sprintf("valor_en_%s_pid_%d", peticion.Direccion, peticion.PID)

		respuesta = RespuestaAcceso{
			Datos: datosMock,
			Exito: true,
		}

		h.Log.DebugContext(ctx, "READ ejecutado exitosamente",
			log.IntAttr("pid", peticion.PID),
			log.StringAttr("direccion", peticion.Direccion),
			log.IntAttr("tamanio", peticion.Tamanio),
			log.StringAttr("datos_leidos", datosMock),
		)

	case "WRITE":
		// Simulación de escritura en memoria
		respuesta = RespuestaAcceso{
			Exito: true,
		}

		h.Log.DebugContext(ctx, "WRITE ejecutado exitosamente",
			log.IntAttr("pid", peticion.PID),
			log.StringAttr("direccion", peticion.Direccion),
			log.StringAttr("datos_escritos", peticion.Datos),
		)

	default:
		respuesta = RespuestaAcceso{
			Exito:   false,
			Mensaje: fmt.Sprintf("Operación no soportada: %s", peticion.Operacion),
		}

		h.Log.WarnContext(ctx, "Operación no soportada",
			log.StringAttr("operacion", peticion.Operacion),
		)
	}

	// Enviar respuesta
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	if err = json.NewEncoder(w).Encode(respuesta); err != nil {
		h.Log.ErrorContext(ctx, "Error al codificar respuesta", log.ErrAttr(err))
		return
	}
}
