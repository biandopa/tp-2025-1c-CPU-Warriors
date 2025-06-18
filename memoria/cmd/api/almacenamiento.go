package api

import (
	"bufio"
	"encoding/json"
	"net/http"
	"os"
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
	)

	if tamanioProceso == "" {
		h.Log.Error("Tamaño del Proceso no proporcionado")
		http.Error(w, "tamaño del oroceso no proporcionado", http.StatusBadRequest)
		return
	}

	if filePath != "" {
		h.Log.DebugContext(ctx, "Archivo de pseudocodigo recibido",
			log.AnyAttr("path-archivo", filePath),
		)
	}

	// Verifica si hay suficiente espacio
	// Inserte función para verificar el espacio disponible

	// Si no hay suficiente espacio, responde con un error
	// Caso contrario, continúa con el procesamiento

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

	// Almacenamos el valor del archivo en el array de instrucciones

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
		// TODO: Que sea un mapa donde la key sea el PID
		h.Instrucciones = append(h.Instrucciones, instruccion)
	}

	// Simulamos una consulta al espacio disponible
	espacioDisponible := 1024 // Simulamos que hay 1024 bytes disponibles

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

	h.Log.InfoContext(ctx, "Consulta de espacio disponible respondida con éxito",
		log.IntAttr("tamaño_disponible", espacioDisponible),
		log.StringAttr("mensaje", response.Mensaje),
	)

	w.WriteHeader(http.StatusOK)
}
