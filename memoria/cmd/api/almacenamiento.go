package api

import (
	"bufio"
	"fmt"
	"math"
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
		// Leemos el PID y el tamaño del proceso de la consulta
		tamanioProceso = r.URL.Query().Get("tamanio-proceso")
		pid            = r.URL.Query().Get("pid")
	)

	if tamanioProceso == "" {
		h.Log.Error("Tamaño del Proceso no proporcionado")
		http.Error(w, "tamaño del proceso no proporcionado", http.StatusBadRequest)
		return
	}

	if pid == "" {
		h.Log.Error("PID no proporcionado")
		http.Error(w, "PID no proporcionado", http.StatusBadRequest)
		return
	}

	//CREAR LA MEMORIA DE USUARIO

	tamanioProcesoInt, _ := strconv.Atoi(tamanioProceso)
	var paginasNecesarias = DivRedondeoArriba(tamanioProcesoInt, h.Config.PageSize)
	var paginasLibres = h.ContarLibres()

	if 0 < paginasLibres-paginasNecesarias {
		h.AsignarMemoriaDeUsuario(paginasNecesarias, pid)
	} else {
		h.Log.Error("No hay espacio disponible")
		return
	}

	// Enviamos la respuesta al kernel
	w.Header().Set("Content-Type", "application/json")
	response := EspacioDisponible{
		Mensaje: "Espacio disponible en memoria",
		Tamaño:  00,
	}

	h.Log.DebugContext(ctx, "Consulta de espacio disponible respondida con éxito",
		log.IntAttr("tamaño_disponible", 00),
		log.StringAttr("mensaje", response.Mensaje),
	)

	w.WriteHeader(http.StatusOK)
}

func DivRedondeoArriba(numerador, denominador int) int {
	return (numerador + denominador - 1) / denominador
}

func (h *Handler) ContarLibres() int {
	libres := 0
	for _, ocupado := range h.FrameTable {
		if !ocupado {
			libres++
		}
	}
	return libres
}

func (h *Handler) AsignarMemoriaDeUsuario(paginasAOcupar int, pid string) {

	var FramesLibres = h.MarcosLibres(paginasAOcupar)

	var tablasPorNivel = h.calcularTablasPorNivel(paginasAOcupar)

	h.Log.Debug("AsignarMemoriaDeUsuario",
		log.AnyAttr("FramesLibres", FramesLibres))

	h.Log.Debug("AsignarMemoriaDeUsuario",
		log.AnyAttr("tablasPorNivel", tablasPorNivel))

	var tablasVacias, _ = h.crearTablasMultinivel(tablasPorNivel, h.Config.EntriesPerPage)

	h.Log.Debug("AsignarMemoriaDeUsuario",
		log.AnyAttr("tablasVacias", tablasVacias))

	h.llenarTablaMultinivel(tablasVacias, FramesLibres, h.Config.EntriesPerPage)

	h.Log.Debug("AsignarMemoriaDeUsuario",
		log.AnyAttr("tablaLLena", tablasVacias))

	//AGREGAR LA TABLA a la TablasProceso

	//HACER LOG
	//  PID: <PID> - Proceso Creado - Tamaño: <TAMAÑO>
}

func (h *Handler) llenarTablasIntermedias(tablas []interface{}, entradasPorTabla int) error {
	niveles := len(tablas)
	if niveles < 2 {
		return nil // No hay niveles intermedios si solo hay 1 nivel (hojas)
	}

	for nivel := niveles - 2; nivel >= 0; nivel-- {
		tablasActuales, ok := tablas[nivel].([]*TablaIntermedia)
		if !ok {
			return fmt.Errorf("nivel %d no es tablas intermedias", nivel)
		}

		siguientes, ok := tablas[nivel+1].([]interface{})
		if !ok {
			// Intentamos castear a []*TablaIntermedia o []*TablaHoja para convertir a []interface{}
			if th, okTh := tablas[nivel+1].([]*TablaHoja); okTh {
				siguientes = make([]interface{}, len(th))
				for i, v := range th {
					siguientes[i] = v
				}
			} else if ti, okTi := tablas[nivel+1].([]*TablaIntermedia); okTi {
				siguientes = make([]interface{}, len(ti))
				for i, v := range ti {
					siguientes[i] = v
				}
			} else {
				return fmt.Errorf("tipo de tabla siguiente nivel desconocido en nivel %d", nivel+1)
			}
		}

		idx := 0
		for _, tabla := range tablasActuales {
			for i := 0; i < entradasPorTabla; i++ {
				if idx < len(siguientes) {
					tabla.Entradas[i] = siguientes[idx]
					idx++
				} else {
					tabla.Entradas[i] = nil
				}
			}
		}
	}
	return nil
}

func (h *Handler) llenarTablaMultinivel(tablas []interface{}, marcos []int, entradasPorTabla int) error {
	niveles := len(tablas)
	if niveles == 0 {
		return fmt.Errorf("no hay tablas")
	}

	tablasHoja, ok := tablas[niveles-1].([]*TablaHoja)
	if !ok {
		return fmt.Errorf("ultimo nivel no es tablas hoja")
	}

	idxMarco := 0
	totalMarcos := len(marcos)

	for _, tablaHoja := range tablasHoja {
		for i := 0; i < entradasPorTabla && idxMarco < totalMarcos; i++ {
			tablaHoja.Entradas[i] = EntradaHoja{
				Marco: marcos[idxMarco],
			}
			idxMarco++
		}
		if idxMarco == totalMarcos {
			break
		}
	}

	if idxMarco < totalMarcos {
		return fmt.Errorf("no hay suficientes entradas para asignar todos los marcos")
	}
	return nil
}

type TablasProceso struct {
	PID             int            `json:"pid"`
	Tamanio         int            `json:"tamanio_proceso"`
	TablasDePaginas []EntradaTabla `json:"tabla_de_paginas"`
}
type EntradaTabla interface{} // puede ser tabla o marco

type TablaIntermedia struct {
	Entradas []EntradaTabla
}

type EntradaHoja struct {
	Marco int
}

type TablaHoja struct {
	Entradas []EntradaHoja
}

func (h *Handler) crearTablasMultinivel(tablasPorNivel []int, entradasPorTabla int) ([]interface{}, error) {
	niveles := len(tablasPorNivel)
	// Usamos []interface{} para guardar tablas de distintos tipos
	tablas := make([]interface{}, niveles)

	for nivel := 0; nivel < niveles; nivel++ {
		cantidadTablas := tablasPorNivel[nivel]

		if nivel == niveles-1 {
			// nivel hoja → tablas hoja
			tablasHoja := make([]*TablaHoja, cantidadTablas)
			for i := 0; i < cantidadTablas; i++ {
				tabla := &TablaHoja{Entradas: make([]EntradaHoja, entradasPorTabla)}
				// Inicializar cada entrada como no asignada
				for j := 0; j < entradasPorTabla; j++ {
					tabla.Entradas[j] = EntradaHoja{
						Marco: -1, // Valor que indica no asignado
					}
				}
				tablasHoja[i] = tabla
			}
			tablas[nivel] = tablasHoja
		} else {
			// nivel intermedio → tablas intermedias
			tablasIntermedias := make([]*TablaIntermedia, cantidadTablas)
			idx := 0
			for i := 0; i < cantidadTablas; i++ {
				tablasIntermedias[i] = &TablaIntermedia{Entradas: make([]EntradaTabla, entradasPorTabla)}

				h.Log.Debug("AsignarMemoriaDeUsuario",
					log.AnyAttr("entradas*CAnt", (entradasPorTabla*cantidadTablas)))

				//aca agregar un if para que no entre si el idx el contador es igual
				//a la cantiddad maxima de entradas necesarias por nivel
				for j := 0; j < (entradasPorTabla); j++ {

					tablasIntermedias[i].Entradas[j] = idx

					h.Log.Debug("AsignarMemoriaDeUsuario",
						log.AnyAttr("tablaNIvel", tablasIntermedias[i].Entradas[j]))
					idx++
				}
			}
			tablas[nivel] = tablasIntermedias

			h.Log.Debug("AsignarMemoriaDeUsuario",
				log.AnyAttr("tablaNIvel", tablas[nivel]))
		}
	}

	return tablas, nil
}

func (h *Handler) calcularTablasPorNivel(paginas int) []int {
	tablasPorNivel := make([]int, h.Config.NumberOfLevels)
	// Nivel hoja (nivel N)
	tablasPorNivel[h.Config.NumberOfLevels-1] = int(math.Ceil(float64(paginas) / float64(h.Config.EntriesPerPage)))

	// Niveles superiores (N-1 ... 1)
	for i := h.Config.NumberOfLevels - 2; i >= 0; i-- {
		tablasPorNivel[i] = int(math.Ceil(float64(tablasPorNivel[i+1]) / float64(h.Config.EntriesPerPage)))
	}

	return tablasPorNivel
}

func (h *Handler) MarcosLibres(paginasNecesarias int) []int {
	libres := []int{}
	for i, ocupado := range h.FrameTable {
		if !ocupado {
			libres = append(libres, i)
			if len(libres) == paginasNecesarias {
				return libres
			}
		}
	}
	return libres
}

func (h *Handler) CargarProcesoEnMemoriaDeSistema(w http.ResponseWriter, r *http.Request) {

	var (
		ctx = r.Context()
		// Leemos el nombre del archivo y el tamaño del proceso de la consulta
		filePath = r.URL.Query().Get("archivo")
		pid      = r.URL.Query().Get("pid")
	)

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

	h.Log.Info("Carga de Proceso en Memoria de Sistema Exitosa")

	w.WriteHeader(http.StatusOK)
}

/*
ejemplo de como queda en este caso : 3 niveles 2 entradas, 5 marcos necesarios
[[{"Entradas":[0,1]}],
	[{"Entradas":[0,1]},{"Entradas":[2, 3]}],
	[{"Entradas":[{"Marco":0},{"Marco":1}]},
		{"Entradas":[{"Marco":2},{"Marco":3}]},
		{"Entradas":[{"Marco":4},{"Marco":-1}]}]]
*/
