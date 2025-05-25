package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

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
func (h *Handler) Fetch(pid int, pc int) (string, error) {
	request := map[string]interface{}{
		"pid": pid,
		"pc":  pc,
	}

	body, _ := json.Marshal(request)
	url := fmt.Sprintf("http://%s:%d/memoria/instruccion", h.Config.IpMemory,
		h.Config.PortMemory)
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(body))
	if err != nil {
		return "", err
	}

	defer func(Body io.ReadCloser) {
		_ = Body.Close()
	}(resp.Body)

	var response struct {
		Instruccion string `json:"instruccion"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return "", err
	}

	h.Log.Info("FETCH realizado", "pid", pid, "pc", pc)

	return response.Instruccion, nil
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

// DECODE
func decode(instruccion string) (string, []string) {
	partes := strings.Fields(instruccion)

	if len(partes) == 0 {
		return "", []string{}
	}

	tipo := strings.ToUpper(partes[0])
	args := partes[1:]

	return tipo, args
}

// EXECUTE
func (h *Handler) Execute(tipo string, args []string, pid int, pc int) (bool, int) {
	nuevoPC := pc
	switch tipo {
	case "NOOP":
		time.Sleep(time.Duration(h.Config.CacheDelay) * time.Millisecond)
		nuevoPC = pc + 1
	case "WRITE":
		/*direccion := args[0]
		datos := args[1]
		dirFisica := traducirDireccion(pid, direccion)
		h.writeMemoria("pid", pid, dirFisica, datos)
		//TODO: implementar traducirDireccion, writeMemoria
		h.Log.Info("ESCRIBIR", pid, dirFisica, datos)
		nuevoPC = pc + 1*/
	case "READ":
		/*direccion, _ := strconv.Atoi(args[0])
		tamanio, _ := strconv.Atoi(args[1])
		dirFisica := traducirDireccion(pid, direccion)
		datoLeido := h.readMemoria(pid, dirFisica, tamanio)
		//TODO: implementar readMemoria
		fmt.Println(datoLeido)
		h.Log.Info("pid", pid, "LEER", dirFisica, datoLeido)
		nuevoPC = pc + 1*/
	case "GOTO":
		nuevoPC, _ = strconv.Atoi(args[0])

	case "IO", "INIT_PROC", "DUMP_MEMORY", "EXIT":
		//h.EnviarProcesoSyscall(pid, tipo, args) //TODO: ver parametros

	default:
		h.Log.Warn("Instrucción no reconocida", log.StringAttr("tipo", tipo))
		nuevoPC = pc + 1
	}

	return true, nuevoPC
}

// CICLO DE INSTRUCCION
func (h *Handler) Ciclo(proceso *Proceso) {
	for {
		fmt.Println("Entre al ciclo")
		instruccion, err := h.Fetch(proceso.PID, proceso.PC)

		if err != nil {
			h.Log.Error("Error en fetch", log.ErrAttr(err))
			return
		}

		tipo, args := decode(instruccion)
		_, nuevoPC := h.Execute(tipo, args, proceso.PID, proceso.PC)
		fmt.Println("Ins", tipo, "Arg", args)
		if tipo == "EXIT" {
			h.Log.Info("Proceso finalizado", "pid", proceso.PID)
			return
		}
		proceso.PC = nuevoPC

		// TODO: Implementar la lógica de interrupciones
	}
}
