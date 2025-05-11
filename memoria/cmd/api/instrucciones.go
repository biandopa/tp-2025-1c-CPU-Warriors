package api

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"strings"

	"github.com/sisoputnfrba/tp-golang/utils/log"
)

func (h *Handler) EnviarInstrucciones(w http.ResponseWriter, r *http.Request) {
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

	url := fmt.Sprintf("http://%s:%d/memoria/instrucciones", h.Config.IpCpu, h.Config.PortCpu)
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(body))
	if err != nil {
		h.Log.Error("Error enviando mensaje",
			slog.Attr{Key: "ip", Value: slog.StringValue(h.Config.IpCpu)},
			slog.Attr{Key: "puerto", Value: slog.IntValue(h.Config.PortCpu)},
			log.ErrAttr(err),
		)
		http.Error(w, "Error enviando mensaje", http.StatusBadRequest)
		return
	}

	if resp != nil {
		defer func() {
			if err = resp.Body.Close(); err != nil {
				h.Log.Error("Error cerrando el cuerpo de la respuesta", log.ErrAttr(err))
			}
		}()

		if resp.StatusCode != http.StatusOK {
			h.Log.Error("Error en la respuesta del CPU",
				log.IntAttr("status_code", resp.StatusCode),
			)
			http.Error(w, "error en la respuesta del CPU", http.StatusInternalServerError)
			return
		}
		h.Log.Info("Mensaje enviado al CPU con éxito",
			log.IntAttr("status_code", resp.StatusCode),
		)
	} else {
		h.Log.Error("Error al enviar mensaje al CPU")
		http.Error(w, "error al enviar mensaje al CPU", http.StatusInternalServerError)
		return
	}

	// Agrego el status Code 200 a la respuesta
	w.WriteHeader(http.StatusOK)

	// Envío la respuesta al cliente con un mensaje de éxito
	_, _ = w.Write([]byte("ok"))
}

func (h *Handler) RecibirInstruccion(w http.ResponseWriter, r *http.Request) {
	// Decode the request body
	var instruccion map[string]interface{}
	err := json.NewDecoder(r.Body).Decode(&instruccion)
	if err != nil {
		h.Log.Error("Error decoding request body", log.ErrAttr(err))
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	h.Log.Info("Instrucción recibida con éxito",
		log.AnyAttr("instruccion", instruccion),
	)

	// Respond with success
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("request processed successfully"))
}

func (h *Handler) RecibirInstrucciones(w http.ResponseWriter, r *http.Request) {
	var (
		ctx = r.Context()
		// Leer tamanioProceso del queryparameter
		tamanioProceso = r.URL.Query().Get("tamanio-proceso")
		// Leer archivoNombre del queryparameter
		pathArchivo = r.URL.Query().Get("path-archivo")
	)

	if tamanioProceso == "" {
		h.Log.Error("Tamaño del Proceso no proporcionado")
		http.Error(w, "tamaño del oroceso no proporcionado", http.StatusBadRequest)
		return
	}

	if pathArchivo == "" {
		h.Log.Error("Nombre del archivo no proporcionado")
		http.Error(w, "nombre del archivo no proporcionado", http.StatusBadRequest)
		return
	}

	h.Log.DebugContext(ctx, "Archivo de pseudocodigo recibido",
		log.AnyAttr("path-archivo", pathArchivo),
	)

	// Verifica si hay suficiente espacio
	// Inserte función para verificar el espacio disponible

	// Si no hay suficiente espacio, responde con un error
	// Caso contrario, continúa con el procesamiento

	// Busca el archivo en el sistema
	file, err := os.OpenFile(h.Config.ScriptsPath+pathArchivo, os.O_RDONLY, os.ModePerm)
	if err != nil {
		h.Log.Error("Error al abrir el archivo de pseudocodigo",
			log.ErrAttr(err),
			log.StringAttr("path-archivo", h.Config.ScriptsPath+pathArchivo),
		)
		http.Error(w, "error al abrir el archivo de pseudocodigo", http.StatusInternalServerError)
		return
	}

	// Nos aseguramos de cerrar el archivo después de usarlo
	defer func() {
		if err = file.Close(); err != nil {
			h.Log.Error("Error al cerrar el archivo de pseudocodigo",
				log.ErrAttr(err),
				log.StringAttr("path-archivo", h.Config.ScriptsPath+pathArchivo),
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
		h.Instrucciones = append(h.Instrucciones, instruccion)
	}

	// Respond with success
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("request processed successfully"))
}
