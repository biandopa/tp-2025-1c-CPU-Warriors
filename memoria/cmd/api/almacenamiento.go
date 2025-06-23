package api

import (
	"bufio"
	"encoding/json"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/sisoputnfrba/tp-golang/utils/log"
)

type EspacioDisponible struct {
	Mensaje string `json:"mensaje"`
	Tamaño  int    `json:"tamaño"`
}

// ConsultarEspacioEInicializar recibe una consulta sobre el espacio libre en memoria.
// En caso de que haya espacio, se inicializa el proceso, se responde con un mensaje de éxito y el tamaño disponible.
// En caso contrario, se responde con un mensaje de error.
// Por el momento, solo responde una respuesta mockeada.
func (h *Handler) ConsultarEspacioEInicializar(w http.ResponseWriter, r *http.Request) {
	var (
		ctx = r.Context()
		// Leemos el nombre del archivo y el tamaño del proceso de la consulta
		filePath       = r.URL.Query().Get("archivo")
		tamanioProceso = r.URL.Query().Get("tamanio-proceso")
		pid            = r.URL.Query().Get("pid")
	)

	if tamanioProceso == "" {
		h.Log.Error("Tamaño del Proceso no proporcionado")
		http.Error(w, "tamaño del proceso no proporcionado", http.StatusBadRequest)
		return
	}

	if filePath == "" {
		h.Log.Error("Archivo de pseudocódigo no proporcionado")
		http.Error(w, "archivo de pseudocódigo no proporcionado", http.StatusBadRequest)
		return
	}

	if pid == "" {
		h.Log.Error("PID no proporcionado")
		http.Error(w, "PID no proporcionado", http.StatusBadRequest)
		return
	}

	pidInt, err := strconv.Atoi(pid)
	if err != nil {
		h.Log.Error("Error al convertir PID a entero",
			log.ErrAttr(err),
			log.StringAttr("pid", pid),
		)
		http.Error(w, "error al convertir PID a entero", http.StatusBadRequest)
		return
	}

	if h.Instrucciones[pidInt] == nil {
		h.Instrucciones[pidInt] = make([]Instruccion, 0)
	}

	// Verifica si hay suficiente espacio
	var espacioDisponible = h.Config.MemorySize
	tamanioProcesoInt, _ := strconv.Atoi(tamanioProceso)

	if 0 < espacioDisponible-tamanioProcesoInt {
		//POSIBLE SEMAFORO ACA!!!! IMPORTANTE
		h.Config.MemorySize = espacioDisponible - tamanioProcesoInt
	} else {
		h.Log.Error("No hay espacio disponible")
		return
	}

	// Busca el archivo en el sistema
	file, err := os.OpenFile(h.Config.ScriptsPath+filePath, os.O_RDONLY, os.ModePerm)
	if err != nil {
		h.Log.Error("Error al abrir el archivo de pseudocodigo",
			log.ErrAttr(err),
			log.StringAttr("path-archivo", h.Config.ScriptsPath+filePath),
		)
		http.Error(w, "error al abrir el archivo de pseudocodigo", http.StatusInternalServerError)
		return
	}

	// Nos aseguramos de cerrar el archivo después de usarlo
	defer func() {
		if err = file.Close(); err != nil {
			h.Log.Error("Error al cerrar el archivo de pseudocodigo",
				log.ErrAttr(err),
				log.StringAttr("path-archivo", h.Config.ScriptsPath+filePath),
			)
		}
	}()

	// Almacenamos el valor del archivo en el mapa de instrucciones

	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		linea := scanner.Text()
		h.Log.DebugContext(ctx, "Leyendo línea del archivo",
			log.StringAttr("linea", linea),
		)

		valores := strings.Split(linea, " ")
		instruccion := Instruccion{
			Instruccion: valores[0],
		}

		if len(valores[1:]) > 0 {
			instruccion.Parametros = valores[1:]
		}

		h.Instrucciones[pidInt] = append(h.Instrucciones[pidInt], instruccion)
	}

	//CREAR LA MEMORIA DE USUARIO

	// Enviamos la respuesta al kernel
	w.Header().Set("Content-Type", "application/json")
	response := EspacioDisponible{
		Mensaje: "Espacio disponible en memoria",
		Tamaño:  espacioDisponible,
	}
	if err = json.NewEncoder(w).Encode(response); err != nil {
		h.Log.ErrorContext(ctx, "Error al codificar la respuesta", log.ErrAttr(err))
		http.Error(w, "Error al codificar la respuesta", http.StatusInternalServerError)
		return
	}

	h.Log.DebugContext(ctx, "Consulta de espacio disponible respondida con éxito",
		log.IntAttr("tamaño_disponible", espacioDisponible),
		log.StringAttr("mensaje", response.Mensaje),
	)

	w.WriteHeader(http.StatusOK)
}
