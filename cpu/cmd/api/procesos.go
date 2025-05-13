package api

import (
	"encoding/json"
	"net/http"

	"github.com/sisoputnfrba/tp-golang/cpu/internal"
	"github.com/sisoputnfrba/tp-golang/utils/log"
)

func (h *Handler) RecibirProcesos(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	// Leer el cuerpo de la solicitud
	decoder := json.NewDecoder(r.Body)
	proceso := &Proceso{}

	h.Log.Debug("Recibi el proceso")

	// Guarda el valor del body en la variable paquete
	err := decoder.Decode(&proceso)
	if err != nil {
		h.Log.ErrorContext(ctx, "Error al decodificar mensaje.", log.ErrAttr(err))
		http.Error(w, "error al decodificar mensaje", http.StatusInternalServerError)
		return
	}

	h.Log.DebugContext(ctx, "Me llego la peticion del Kernel",
		log.AnyAttr("paquete", proceso),
	)
	// Agregar ciclo de instrucción
	go func() {
		nuevoPC := h.ciclo(proceso)
		proceso.PC = nuevoPC
	}()
	// TODO: Agregar ejecución de instrucción
	// Añadir la syscall
	syscall := &internal.ProcesoSyscall{
		PID:         proceso.PID,
		PC:          proceso.PC,
		Instruccion: "EXIT",     // Ejemplo de instrucción mockeado
		Args:        []string{}, // Ejemplo de argumentos mockeados
	}
	go func() {
		err = h.Service.EnviarProcesoSyscall(ctx, syscall)
	}()

	// Agrego el status Code 200 a la respuesta
	w.WriteHeader(http.StatusOK)

	// Envío la respuesta al cliente con un mensaje de éxito
	_, _ = w.Write([]byte("ok"))
}
