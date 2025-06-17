package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/sisoputnfrba/tp-golang/cpu/internal"
	"github.com/sisoputnfrba/tp-golang/utils/log"
)

func (h *Handler) RecibirInstrucciones(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	// Leer el cuerpo de la solicitud
	decoder := json.NewDecoder(r.Body)
	paquete := map[string]interface{}{}

	// Guarda el valor del body en la variable paquete
	err := decoder.Decode(&paquete)
	if err != nil {
		h.Log.ErrorContext(ctx, "Error al decodificar mensaje", log.ErrAttr(err))
		http.Error(w, "error al decodificar mensaje", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("ok"))
}

func (h *Handler) EnviarInstruccion(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	// Creo instruccion
	instruccion := map[string]interface{}{
		"tipo": "instruccion",
		"datos": map[string]interface{}{
			"codigo": "codigo de la instruccion",
		},
	}

	// Conviero la estructura del proceso a un []bytes (formato en el que se envían las peticiones)
	body, err := json.Marshal(instruccion)
	if err != nil {
		h.Log.ErrorContext(ctx, "Error codificando mensaje", log.ErrAttr(err))
		http.Error(w, "Error codificando mensaje", http.StatusBadRequest)
		return
	}

	url := fmt.Sprintf("http://%s:%d/cpu/instruccion", h.Config.IpMemory, h.Config.PortMemory)
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(body))
	if err != nil {
		h.Log.ErrorContext(ctx, "Error enviando mensaje",
			log.StringAttr("ip", h.Config.IpMemory),
			log.IntAttr("puerto", h.Config.PortMemory),
			log.ErrAttr(err),
		)
		http.Error(w, "Error enviando mensaje", http.StatusBadRequest)
		return
	}

	if resp != nil {
		h.Log.Debug("Respuesta del servidor",
			log.StringAttr("status", resp.Status),
		)
	} else {
		h.Log.Debug("Respuesta del servidor: nil")
	}

	// Agrego el status Code 200 a la respuesta
	w.WriteHeader(http.StatusOK)

	// Envío la respuesta al cliente con un mensaje de éxito
	_, _ = w.Write([]byte("ok"))
}

// FETCH
func (h *Handler) Fetch(pid int, pc int) (Instruccion, error) {
	instruccion, err := h.Memoria.FetchInstruccion(pid, pc)
	if err != nil {
		return Instruccion{}, err
	}

	// Convertir de memoria.Instruccion a api.Instruccion
	response := Instruccion{
		Instruccion: instruccion.Instruccion,
		Parametros:  instruccion.Parametros,
	}

	h.Log.Info("FETCH realizado", "pid", pid, "pc", pc)

	return response, nil
}

// Fetch De prueba hasta tener hecho memoria
/*
func (h *Handler) Fetch(pid int, pc int) (string, error) {
	mockInstrucciones := []string{
		"NOOP",
		"WRITE 100 42",
		"READ 100 4",
		"GOTO 3",
		"EXIT",
	}

	if pc < len(mockInstrucciones) {
		instruccion := mockInstrucciones[pc]
		h.Log.Info("FETCH mockeado", "pid", pid, "pc", pc, "instruccion", instruccion)
		return instruccion, nil
	}

	h.Log.Warn("PC fuera de rango de instrucciones mock", "pc", pc)
	return "EXIT", nil
}*/

// Decode Interpreta la instrucción y sus argumentos. Además, verifica si la misma requiere de una
// traducción de dirección lógica a física.
func decode(instruccion Instruccion) (string, []string) {
	//TODO: Implementar la parte de la traducción de dirección lógica a física.
	tipo := strings.ToUpper(instruccion.Instruccion)
	args := instruccion.Parametros

	return tipo, args
}

// EXECUTE
func (h *Handler) Execute(tipo string, args []string, pid, pc int) (bool, int) {
	var (
		nuevoPC       = pc
		returnControl bool
	)

	switch tipo {
	case "NOOP":
		time.Sleep(time.Duration(h.Config.CacheDelay) * time.Millisecond)
		nuevoPC = pc + 1
	case "WRITE":
		if len(args) < 2 {
			h.Log.Error("WRITE requiere al menos 2 argumentos: dirección y datos",
				log.IntAttr("pid", pid),
				log.IntAttr("pc", pc))
			return false, pc
		}
		direccion := args[0]
		datos := args[1]

		// TODO: Por ahora usamos la dirección lógica directamente (implementar traducción)
		dirFisica := direccion

		if err := h.Memoria.Write(pid, dirFisica, datos); err != nil {
			h.Log.Error("Error al escribir en memoria",
				log.ErrAttr(err),
				log.IntAttr("pid", pid),
				log.StringAttr("direccion", dirFisica),
				log.StringAttr("datos", datos))
			return false, pc
		}

		h.Log.Debug("WRITE ejecutado exitosamente",
			log.IntAttr("pid", pid),
			log.StringAttr("direccion_fisica", dirFisica),
			log.StringAttr("datos", datos))
		nuevoPC = pc + 1

	case "READ":
		if len(args) < 2 {
			h.Log.Error("READ requiere al menos 2 argumentos: dirección y tamaño",
				log.IntAttr("pid", pid),
				log.IntAttr("pc", pc))
			return false, pc
		}
		direccion := args[0]
		tamanio, err := strconv.Atoi(args[1])
		if err != nil {
			h.Log.Error("Tamaño inválido en instrucción READ",
				log.ErrAttr(err),
				log.IntAttr("pid", pid),
				log.StringAttr("tamanio_str", args[1]))
			return false, pc
		}

		// TODO: Por ahora usamos la dirección lógica directamente (implementar traducción)
		dirFisica := direccion

		datoLeido, err := h.Memoria.Read(pid, dirFisica, tamanio)
		if err != nil {
			h.Log.Error("Error al leer de memoria",
				log.ErrAttr(err),
				log.IntAttr("pid", pid),
				log.StringAttr("direccion", dirFisica),
				log.IntAttr("tamanio", tamanio))
			return false, pc
		}

		h.Log.Debug("READ ejecutado exitosamente",
			log.IntAttr("pid", pid),
			log.StringAttr("direccion_fisica", dirFisica),
			log.StringAttr("dato_leido", datoLeido),
			log.IntAttr("tamanio", tamanio))
		nuevoPC = pc + 1

	case "GOTO":
		pcAtoi, err := strconv.Atoi(args[0])
		if err != nil {
			h.Log.Error("GOTO requiere un argumento numérico válido",
				log.ErrAttr(err),
				log.IntAttr("pid", pid),
				log.StringAttr("argumento", args[0]))
			return false, pc
		}
		nuevoPC = pcAtoi

	case "IO", "INIT_PROC", "DUMP_MEMORY", "EXIT":
		syscall := &internal.ProcesoSyscall{
			PID:         pid,
			PC:          pc + 1, // Avanzamos el PC para la syscall
			Instruccion: tipo,
			Args:        args,
		}

		if err := h.Service.EnviarProcesoSyscall(syscall); err != nil {
			h.Log.Error("Error al enviar proceso syscall", log.ErrAttr(err))
			return false, pc // Si hay error, no avanzamos el PC
		}

		h.Log.Debug("Syscall enviada al kernel",
			log.IntAttr("pid", pid),
			log.StringAttr("instruccion", tipo),
			log.IntAttr("pc_nuevo", pc+1))

		// Para syscalls, retornamos false para indicar que el CPU debe devolver el control al kernel
		returnControl = true
		nuevoPC = pc + 1 // Avanzamos el PC para la syscall

	default:
		h.Log.Warn("Instrucción no reconocida", log.StringAttr("tipo", tipo))
		nuevoPC = pc + 1
	}

	return returnControl, nuevoPC
}

// CICLO DE INSTRUCCION
func (h *Handler) Ciclo(proceso *Proceso) int {
	for {
		h.Log.Debug("Iniciando ciclo de instrucción",
			log.IntAttr("pid", proceso.PID),
			log.IntAttr("pc", proceso.PC))

		instruccion, err := h.Fetch(proceso.PID, proceso.PC)
		if err != nil {
			h.Log.Error("Error en fetch", log.ErrAttr(err))
			return proceso.PC // Si hay error, no avanzamos el PC
		}

		tipo, args := decode(instruccion)
		h.Log.Debug("Instrucción decodificada",
			log.StringAttr("tipo", tipo),
			log.AnyAttr("args", args))

		continuar, nuevoPC := h.Execute(tipo, args, proceso.PID, proceso.PC)
		proceso.PC = nuevoPC

		// Si Execute retorna false, significa que hay que devolver el control al kernel
		// (por ejemplo, por una syscall o error)
		if !continuar {
			h.Log.Debug("Devolviendo control al kernel",
				log.IntAttr("pid", proceso.PID),
				log.StringAttr("razon", tipo),
				log.IntAttr("pc_final", nuevoPC))
			return proceso.PC
		}

		// Si la instrucción es EXIT, finalizamos el ciclo
		if tipo == "EXIT" {
			h.Log.Debug("Proceso finalizado por EXIT",
				log.IntAttr("pid", proceso.PID),
				log.IntAttr("pc_final", proceso.PC))
			return proceso.PC
		}

		// TODO: Implementar la lógica de interrupciones
	}
}
