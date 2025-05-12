package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
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
func (h *Handler) fetch(pid int, pc int) (string, error) {
request := map[string]interface{}{
"pid": pid,
"pc": pc,
}

body, _ := json.Marshal(request)
url := fmt.Sprintf("http://%s:%d/memoria/instruccion", h.Config.IpMemory,
h.Config.PortMemory)
resp, err := http.Post(url, "application/json", bytes.NewBuffer(body))
if err != nil {
return "", err
}
defer resp.Body.Close()

var response struct {
instruccion string `json:&quot;instruccion&quot;`
}

if err := json.NewDecoder(resp.Body).Decode(&amp;response); err != nil {
return "";, err
}
h.log.Info(pid, "FETCH", pc)
return response.Instruccion, nil
}
//DECODE
func decode(instruccion string) (string, []string){
	partes :=strings.Fields(instruccion)

	if len(partes) == 0 {
		return "", []string{}
	}

	tipo := strings.ToUpper(partes[0])
	args := partes[1:]

	return tipo, args
}
// EXECUTE
func (h *Handler) execute(tipo string, args []string, pid int) (bool, int) {
	switch tipo {
	case "NOOP":
	time.Sleep(h.Config.CacheDelay * time.Millisecond)
	nuevoPC = incrementarPC()
	case "WRITE":
	direccion := args[0]
	datos := args[1]
	dirFisica := traducirDireccion(pid, direccion)
	h.writeMemoria(pid, dirFisica, datos)
	//TODO: implementar traducirDireccion, writeMemoria
	h.log.String(pid,"ESCRIBIR", dirFisica, datos)
	nuevoPC = incrementarPC()
	case "READ":
	direccion := args[0]
	tamanio := args[1]
	dirFisica := traducirDireccion(pid, direccion)
	datoLeido = h.readMemoria(pid, dirFisica, tamanio)
	//TODO: implementar readMemoria
	fmt.printf(datoLeido)
	h.log.String(pid,"LEER", dirFisica, datoLeido)
	nuevoPC = incrementarPC()
	case "GOTO":
		saltarAPC()
	
	case "IO","INIT_PROC", "DUMP_MEMORY", "EXIT":
	h.EnviarProcesoSyscall(pid, tipo, args) //TODO: ver parametros
	
	default:
	h.Log.Warn("Instrucción no reconocida", log.String("tipo", tipo))
	nuevoPC = incrementarPC()
	}
	
	
	//TODO: Implementar incrementarPC
	return true, nuevoPC
	}