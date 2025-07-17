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

// Fetch obtiene la instrucción de memoria para un proceso dado (pid) y contador de programa (pc).
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

	//Log obligatorio: Fetch instrucción
	//“## PID: <PID> - FETCH - Program Counter: <PROGRAM_COUNTER>”
	h.Log.Info(fmt.Sprintf("## PID: %d - FETCH - Program Counter: %d", pid, pc))

	return response, nil
}

// Decode Interpreta la instrucción y sus argumentos. Además, verifica si la misma requiere de una
// traducción de dirección lógica a física.
func (h *Handler) decode(instruccion Instruccion) (string, []string, error) {
	tipo := strings.ToUpper(instruccion.Instruccion)
	args := instruccion.Parametros

	// Para READ y WRITE, no traducimos aquí - lo harán las funciones LeerConCache y EscribirConCache
	// que necesitan la dirección lógica original para calcular el número de página correctamente

	return tipo, args, nil
}

// Execute ejecuta la instrucción decodificada. Dependiendo del tipo de instrucción, puede
// requerir interacción con la memoria, el kernel o simplemente ser una operación no operativa (NOOP).
func (h *Handler) Execute(tipo string, args []string, pid, pc int) (bool, int) {
	var (
		nuevoPC       = pc
		returnControl bool // Retornamos false para indicar que el CPU debe devolver el control al kernel
	)

	switch tipo {
	case "NOOP":
		time.Sleep(time.Duration(h.Config.CacheDelay) * time.Millisecond)
		nuevoPC++
		returnControl = true

	case "WRITE":
		if len(args) < 2 {
			h.Log.Error("WRITE requiere al menos 2 argumentos: dirección y datos",
				log.IntAttr("pid", pid),
				log.IntAttr("pc", pc))
			return false, pc
		}
		direccionLogica := args[0] // Dirección lógica
		datos := args[1]

		// Usar la MMU para escribir con caché. Si la caché no está habilitada, se escribe directamente en memoria
		direccionFisica, err := h.Service.MMU.EscribirConCache(pid, direccionLogica, datos)
		if err != nil {
			h.Log.Error("Error al escribir en memoria",
				log.ErrAttr(err),
				log.IntAttr("pid", pid),
				log.StringAttr("direccion", direccionLogica),
				log.StringAttr("datos", datos))
			return false, pc
		}

		//Log obligatorio: Lectura/Escritura Memoria
		//“PID: <PID> - Acción: <LEER / ESCRIBIR> - Dirección Física: <DIRECCION_FISICA> - Valor: <VALOR LEIDO / ESCRITO>”.
		//Log obligatorio: Lectura/Escritura Memoria
		//"PID: <PID> - Acción: <LEER / ESCRIBIR> - Dirección Física: <DIRECCION_FISICA> - Valor: <VALOR LEIDO / ESCRITO>".
		h.Log.Info(fmt.Sprintf("## PID: %d - Acción: ESCRIBIR - Dirección Física: %s - Valor: %s",
			pid, direccionFisica, datos))

		nuevoPC++
		returnControl = true

	case "READ":
		if len(args) < 2 {
			h.Log.Error("READ requiere al menos 2 argumentos: dirección y tamaño",
				log.IntAttr("pid", pid),
				log.IntAttr("pc", pc))
			return false, pc
		}

		direccionLogica := args[0] // Dirección lógica
		tamanio, err := strconv.Atoi(args[1])
		if err != nil {
			h.Log.Error("Tamaño inválido en instrucción READ",
				log.ErrAttr(err),
				log.IntAttr("pid", pid),
				log.StringAttr("tamanio_str", args[1]))
			return false, pc
		}

		// Usar la MMU para leer con caché. Si la caché no está habilitada, se lee directamente en memoria
		var direccionFisica string
		var datoLeido string
		datoLeido, direccionFisica, err = h.Service.MMU.LeerConCache(pid, direccionLogica, tamanio)
		if err != nil {
			h.Log.Error("Error al leer de memoria",
				log.ErrAttr(err),
				log.IntAttr("pid", pid),
				log.StringAttr("direccion", direccionLogica),
				log.IntAttr("tamanio", tamanio))
			return false, pc
		}

		//Log obligatorio: Lectura/Escritura Memoria
		//“PID: <PID> - Acción: <LEER / ESCRIBIR> - Dirección Física: <DIRECCION_FISICA> - Valor: <VALOR LEIDO / ESCRITO>”.

		h.Log.Info(fmt.Sprintf("## PID: %d - Acción: LEER - Dirección Física: %s - Valor: %s",
			pid, direccionFisica, datoLeido))

		nuevoPC++
		returnControl = true

	case "GOTO":
		if len(args) != 1 {
			h.Log.Error("GOTO requiere un único argumento numérico",
				log.IntAttr("pid", pid),
				log.IntAttr("pc", pc),
				log.AnyAttr("args", args))
			return false, pc
		}

		pcAtoi, err := strconv.Atoi(args[0])
		if err != nil {
			h.Log.Error("GOTO requiere un argumento numérico válido",
				log.ErrAttr(err),
				log.IntAttr("pid", pid),
				log.StringAttr("argumento", args[0]))
			return false, pc
		}
		nuevoPC = pcAtoi
		returnControl = true
	case "INIT_PROC":
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

		// Para syscalls, retornamos false para indicar que el CPU debe devolver el control al kernel (menos INIT_PROC)
		returnControl = true
		nuevoPC++ // Avanzamos el PC para la syscall

	case "IO", "DUMP_MEMORY", "EXIT":
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
		returnControl = false
		nuevoPC++ // Avanzamos el PC para la syscall

	default:
		h.Log.Debug("Instrucción no reconocida", log.StringAttr("tipo", tipo))
		return returnControl, nuevoPC
	}

	// Log obligatorio: Instrucción Ejecutada
	//“## PID: <PID> - Ejecutando: <INSTRUCCION> - <PARAMETROS>”.
	h.Log.Info(fmt.Sprintf("## PID: %d - Ejecutando: %s - %s", pid, tipo, strings.Join(args, " ")))

	return returnControl, nuevoPC
}

// Ciclo ejecuta un ciclo de instrucciones para un proceso dado. Su retorno es un mensaje de error
// si ocurre algún problema.
func (h *Handler) Ciclo(proceso *Proceso) string {
	for {
		h.Log.Debug("Iniciando ciclo de instrucción",
			log.IntAttr("pid", proceso.PID),
			log.IntAttr("pc", proceso.PC),
		)

		instruccion, err := h.Fetch(proceso.PID, proceso.PC)
		if err != nil {
			h.Log.Error("Error en fetch", log.ErrAttr(err))
			return fmt.Sprintf("Error en fetch: %v", err)
		}

		tipo, args, err := h.decode(instruccion)
		if err != nil {
			h.Log.Error("Error en decodificación", log.ErrAttr(err))
			return fmt.Sprintf("Error en decodificación: %v", err)
		}
		h.Log.Debug("Instrucción decodificada",
			log.StringAttr("tipo", tipo),
			log.AnyAttr("args", args))

		// Ejecutar instrucción
		continuar, nuevoPC := h.Execute(tipo, args, proceso.PID, proceso.PC)
		proceso.PC = nuevoPC

		// Si la instrucción es EXIT, finalizamos el proceso
		if tipo == "EXIT" {
			h.Log.Debug("Proceso finalizado",
				log.IntAttr("pid", proceso.PID))

			// Limpiar memoria (TLB y caché) cuando el proceso termina
			h.Service.LimpiarMemoriaProceso(proceso.PID)

			break
		}

		// Si no se debe continuar (por syscall bloqueante), salir del ciclo
		if !continuar {
			h.Log.Debug("Proceso pausado por syscall",
				log.IntAttr("pid", proceso.PID))
			break
		}

		// Verificar interrupciones después de cada instrucción
		if h.Service.HayInterrupciones() {
			h.Log.Debug("Interrupción detectada, saliendo del ciclo de instrucción",
				log.IntAttr("pid", proceso.PID),
				log.IntAttr("pc", proceso.PC),
			)

			// Obtener la interrupción para verificar si es de desalojo
			interrupcion, _ := h.Service.ObtenerInterrupcion()
			if interrupcion.Tipo == internal.InerrupcionDesalojo {
				h.Log.Debug("Interrupción de desalojo detectada, limpiando memoria",
					log.IntAttr("pid", proceso.PID))
				h.Service.LimpiarMemoriaProceso(proceso.PID)
			}

			return "Interrupción detectada, proceso pausado"
		}
	}

	h.Log.Debug("Ciclo de instrucción completado",
		log.IntAttr("pid", proceso.PID),
		log.IntAttr("pc", proceso.PC))
	return "Proceso ejecutado exitosamente"
}
