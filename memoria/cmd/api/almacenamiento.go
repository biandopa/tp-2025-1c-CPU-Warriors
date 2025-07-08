package api

import (
	"bufio"
	"fmt"
	"io"
	"math"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

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
		h.AsignarMemoriaDeUsuario(paginasNecesarias, pid, false)
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

// tamanioDelProcseo / sizePAge = catnidadDeMarcos
// cantidadMarcos/PAginas

func (h *Handler) AsignarMemoriaDeUsuario(paginasAOcupar int, pid string, esActualizacion bool) {

	var FramesLibres = h.MarcosLibres(paginasAOcupar)

	var tablasPorNivel = h.calcularTablasPorNivel(paginasAOcupar)

	var paginasPorNivel = h.calcularEntradasPorNivel(paginasAOcupar)

	h.Log.Debug("AsignarMemoriaDeUsuario",
		log.AnyAttr("FramesLibres", FramesLibres))

	h.Log.Debug("AsignarMemoriaDeUsuario",
		log.AnyAttr("tablasPorNivel", tablasPorNivel))

	h.Log.Debug("AsignarMemoriaDeUsuario",
		log.AnyAttr("paginasPorNivel", paginasPorNivel))

	var tablasVacias, _ = h.crearTablasMultinivel(tablasPorNivel, paginasPorNivel, h.Config.EntriesPerPage)

	h.Log.Debug("AsignarMemoriaDeUsuario",
		log.AnyAttr("tablasVacias", tablasVacias))

	h.llenarTablaMultinivel(tablasVacias, FramesLibres, h.Config.EntriesPerPage)

	h.Log.Debug("AsignarMemoriaDeUsuario",
		log.AnyAttr("tablaLLena", tablasVacias))

	//AGREGAR LA TABLA a la TablasProceso

	var tablaProceso *TablasProceso

	if esActualizacion {
		for _, tp := range h.TablasProcesos {
			if tp.PID == pid {
				tp.TablasDePaginas = tablasVacias
			}
		}
	} else {
		tablaProceso = &TablasProceso{
			PID:             pid,
			Tamanio:         paginasAOcupar * h.Config.MemorySize,
			TablasDePaginas: tablasVacias,
		}
	}

	h.Log.Debug("tablaProceso",
		log.AnyAttr("tablaProceso", tablaProceso))

	h.TablasProcesos = append(h.TablasProcesos, tablaProceso)

	h.Log.Debug("TablasProcesos",
		log.AnyAttr("TablasProcesos", h.TablasProcesos))
	//HACER LOG
	//  PID: <PID> - Proceso Creado - Tamaño: <TAMAÑO>

	//
	//BORRAR DE ACA PARA ABAJP ESTA AHORA PARA PROBAR LAS COSAS

	if esActualizacion == false {

		//copy(h.EspacioDeUsuario[0:], []byte("hola"))

		tabla := h.CrearTabla(h.Config.NumberOfLevels, h.Config.EntriesPerPage)

		//lectura, _ := h.leerValor(tabla, []int{0, 0, 1})

		h.LlenarTablaConValores(tabla, []int{1, 2, 3})

		h.Log.Debug("leerValor",
			log.AnyAttr("lectura", tabla))

		//h.PasarProcesoASwapAuxiliar(pid)

		//h.LeerPagina(0, 2, 1, pid)

		//h.EscribirPagina(marco int, offset int, valorAEscribir string, pid string)
		//h.EscribirPagina(1, 0, "ey", pid)

		//h.LeerPagina(1, 0, 5, pid)

		/*h.Log.Debug("FinalizarProcesoFuncionAuxiliar",
		log.AnyAttr("TablasProcesos", h.TablasProcesos))
		*/
		//h.FinalizarProcesoFuncionAuxiliar(pid)

		//h.SacarProcesoDeSwap(pid)

		//h.DumpProcesoFuncionAuxiliar(pid)

		//h.BuscarMarcoPorPagina([]int{0, 0, 1}, pid)

	}

}

func (h *Handler) escribirMarcoEnSwap(archivo *os.File, marco int) error {
	// Calculamos la posición en bytes donde va este marco en swap.bin
	posicion := int64(marco * h.Config.PageSize)

	//La poscion exacta la saca de la lista
	// Seek a la posición exacta
	_, err := archivo.Seek(posicion, 0)
	if err != nil {
		return fmt.Errorf("error haciendo seek para marco %d: %w", marco, err)
	}

	// Tomamos el fragmento correspondiente del slice memoria
	inicio := marco * h.Config.PageSize
	fin := inicio + h.Config.PageSize

	// Escribimos ese bloque en el archivo
	n, err := archivo.Write(h.EspacioDeUsuario[inicio:fin])
	if err != nil {
		return fmt.Errorf("error escribiendo marco %d: %w", marco, err)
	}
	if n != h.Config.PageSize {
		return fmt.Errorf("error: bytes escritos %d no coincide con sizePage %d para marco %d", n, h.Config.PageSize, marco)
	}

	return nil
}

func (h *Handler) ObtenerMarcosDeLaTabla(tablas []interface{}) []int {
	marcos := []int{}

	for _, nivel := range tablas {
		switch t := nivel.(type) {
		case []*TablaIntermedia:
			for _, tablaInter := range t {
				for _, entrada := range tablaInter.Entradas {
					if entrada != nil {
						// entrada es interface{}, puede ser *TablaIntermedia o *TablaHoja
						switch subtabla := entrada.(type) {
						case *TablaIntermedia:
							marcos = append(marcos, h.ObtenerMarcosDeLaTabla([]interface{}{[]*TablaIntermedia{subtabla}})...)
						case *TablaHoja:
							marcos = append(marcos, h.ObtenerMarcosDeLaTabla([]interface{}{[]*TablaHoja{subtabla}})...)
						}
					}
				}
			}

		case []*TablaHoja:
			for _, tablaHoja := range t {
				for _, entradaHoja := range tablaHoja.Entradas {
					if entradaHoja.Marco != -1 {
						marcos = append(marcos, entradaHoja.Marco)
					}
				}
			}
		}
	}

	return marcos
}

func (h *Handler) BuscarProcesoPorPID(pid string) (*TablasProceso, error) {
	for _, proceso := range h.TablasProcesos {
		if proceso.PID == pid {
			return proceso, nil
		}
	}
	return nil, fmt.Errorf("proceso con PID %s no encontrado", pid)
}

//hacer una funcion que se llame cambiarASwap
//

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

/*type TablasProcesos struct {
	TablasProcesos []*TablasProceso `json:"TablasProcesos"`
}*/

type TablasProceso struct {
	PID             string        `json:"pid"`
	Tamanio         int           `json:"tamanio_proceso"`
	TablasDePaginas []interface{} `json:"tabla_de_paginas"`
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

func (h *Handler) crearTablasMultinivel(tablasPorNivel []int, paginasPorNivel []int, entradasPorTabla int) ([]interface{}, error) {
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

					if paginasPorNivel[nivel] > idx {

						h.Log.Debug("AsignarMemoriaDeUsuario",
							log.AnyAttr("paginasPornivel", paginasPorNivel[nivel]))
						tablasIntermedias[i].Entradas[j] = idx

						h.Log.Debug("AsignarMemoriaDeUsuario",
							log.AnyAttr("tablaNIvel", tablasIntermedias[i].Entradas[j]))
						idx++
					}
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

func (h *Handler) calcularEntradasPorNivel(paginas int) []int {
	entradasPorNivel := make([]int, h.Config.NumberOfLevels)

	// Nivel hoja (nivel más bajo)
	entradasPorNivel[h.Config.NumberOfLevels-1] = paginas

	// Niveles superiores
	for i := h.Config.NumberOfLevels - 2; i >= 0; i-- {
		// Cada tabla de este nivel apunta a la del siguiente nivel,
		// así que necesitamos una entrada por tabla del nivel inferior
		entradasPorNivel[i] = int(math.Ceil(float64(entradasPorNivel[i+1]) / float64(h.Config.EntriesPerPage)))
	}

	return entradasPorNivel
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

func (h *Handler) PasarProcesoASwap(w http.ResponseWriter, r *http.Request) {

	var (
		//ctx = r.Context()
		// Leemos el PID
		pid = r.URL.Query().Get("pid")
	)

	if pid == "" {
		h.Log.Error("PID no proporcionado")
		http.Error(w, "PID no proporcionado", http.StatusBadRequest)
		return
	}

	h.PasarProcesoASwapAuxiliar(pid)

}

func (h *Handler) PasarProcesoASwapAuxiliar(pid string) {

	procesYTablaAsociada, _ := h.BuscarProcesoPorPID(pid)
	h.Log.Debug("PasarProcesoASwapAuxiliar",
		log.AnyAttr("procesYTablaAsociada", procesYTablaAsociada.TablasDePaginas))

	marcosDelProceso := h.ObtenerMarcosDeLaTabla(procesYTablaAsociada.TablasDePaginas)

	h.Log.Debug("PasarProcesoASwapAuxiliar",
		log.AnyAttr("ObtenerMarcosValidos", h.EspacioDeUsuario))

	h.Log.Debug("PasarProcesoASwapAuxiliar",
		log.AnyAttr("ObtenerMarcosValidos", h.EspacioDeUsuario))

	h.Log.Debug("PasarProcesoASwapAuxiliar",
		log.AnyAttr("ObtenerMarcosValidos", marcosDelProceso))

	//iterar la lista de marcos un for, y por cada uno multiplicarlo por el sizepage

	archivoSwap, err := os.OpenFile("/home/utnso/Desktop/tp-2025-1c-CPU-Warriors/memoria/swapfile.bin", os.O_RDWR, 0644)
	if err != nil {
		panic(err)
	}
	//Para cerrarlo despues
	defer archivoSwap.Close()

	pidInt, _ := strconv.Atoi(pid)

	h.Log.Debug("PasarProcesoASwapAuxiliar",
		log.AnyAttr("ProcesoPorPosicionSwap", pidInt))

	for i := 0; i < len(marcosDelProceso); i++ {
		h.ProcesoPorPosicionSwap = append(h.ProcesoPorPosicionSwap, pidInt)
	}

	h.Log.Debug("PasarProcesoASwapAuxiliar",
		log.AnyAttr("ProcesoPorPosicionSwap", h.ProcesoPorPosicionSwap))

	for marco := range marcosDelProceso {
		err := h.escribirMarcoEnSwap(archivoSwap, marco)
		if err != nil {
			panic(err)
		}

		copy(h.EspacioDeUsuario[marco*h.Config.PageSize:((marco+1)*h.Config.PageSize-1)], make([]byte, h.Config.PageSize))
	}

	h.Log.Debug("PasarProcesoASwapAuxiliar",
		log.AnyAttr("EspacioDeUsuario", h.EspacioDeUsuario))

}

func (h *Handler) SacarProcesoDeSwap(pid string) {

	//lista cada elemento un proceso, donde se un id, osea cada marco un id {0,0,0,0,0}
	// proceso 1 {0,0,0,0,0,1,1,1}
	// sale proceso 0 borrar los ceros {1,1,1}

	//a nivel swap
	// proceso 1 {0,0,0,0,0,1,1,1},
	// sale proceso 0 borrar los ceros {1,1,1}, borrar los bits posciones del proceso * pagesize

	//DESUSPENSION RECIBIMOS EL PID
	// buscar posiciones en el swap

	pidDeSwap, _ := strconv.Atoi(pid)

	h.Log.Debug("SacarProcesoDeSwap",
		log.AnyAttr("pidDeSwap", pidDeSwap))

	posicionEnSwap := h.PosicionesDeProcesoEnSwap(pidDeSwap)

	h.Log.Debug("SacarProcesoDeSwap",
		log.AnyAttr("posicionEnSwap", posicionEnSwap))

	h.Log.Debug("SacarProcesoDeSwap",
		log.AnyAttr("lenPosicionEnSwap", len(posicionEnSwap)))

	var paginasNecesarias = len(posicionEnSwap)
	var paginasLibres = h.ContarLibres()

	h.Log.Debug("SacarProcesoDeSwap",
		log.AnyAttr("paginasNecesarias", paginasNecesarias))

	h.Log.Debug("SacarProcesoDeSwap",
		log.AnyAttr("paginasNecesarias", paginasLibres))

	if 0 < paginasLibres-paginasNecesarias {
		//agregar un parametro que reprsente si es actualizacion o no
		h.AsignarMemoriaDeUsuario(paginasNecesarias, pid, true)
	} else {
		h.Log.Error("No hay espacio disponible")
		return
	}

	procesYTablaAsociadaDeSwap, _ := h.BuscarProcesoPorPID(pid)
	h.Log.Debug("SacarProcesoDeSwap",
		log.AnyAttr("procesYTablaAsociada", procesYTablaAsociadaDeSwap.TablasDePaginas))

	marcosDelProcesoDeSwap := h.ObtenerMarcosDeLaTabla(procesYTablaAsociadaDeSwap.TablasDePaginas)
	h.Log.Debug("SacarProcesoDeSwap",
		log.AnyAttr("marcosDelProcesoDeSwap", marcosDelProcesoDeSwap))

	//ir escribiendo cada frame en memoria
	h.CargarPaginasEnMemoriaDesdeSwap(posicionEnSwap, marcosDelProcesoDeSwap)
	h.Log.Debug("SacarProcesoDeSwap",
		log.AnyAttr("CargarPaginasEnMemoriaDesdeSwap", h.EspacioDeUsuario))

	//compactar la posicion en swap y borrarlo en la lista de procesos}
	h.eliminarOcurrencias(pidDeSwap)
	h.Log.Debug("SacarProcesoDeSwap",
		log.AnyAttr("eliminarOcurrencias", h.ProcesoPorPosicionSwap))

	h.CompactarSwap()

}

func (h *Handler) CompactarSwap() error {
	pageSize := h.Config.PageSize

	// Abrimos el archivo en modo lectura/escritura
	swapFile, err := os.OpenFile("/home/utnso/Desktop/tp-2025-1c-CPU-Warriors/memoria/swapfile.bin", os.O_RDWR, 0644)
	if err != nil {
		return fmt.Errorf("error abriendo swap.bin: %w", err)
	}
	defer swapFile.Close()

	posDestino := 0
	nuevaPosiciones := make([]int, 0)

	for _, marco := range h.ProcesoPorPosicionSwap {
		offset := marco * pageSize
		buffer := make([]byte, pageSize)

		// Leer el marco original
		_, err := swapFile.ReadAt(buffer, int64(offset))
		if err != nil && err != io.EOF {
			return fmt.Errorf("error leyendo marco %d: %w", marco, err)
		}

		// Escribirlo en la nueva posición
		_, err = swapFile.WriteAt(buffer, int64(posDestino))
		if err != nil {
			return fmt.Errorf("error escribiendo marco en pos %d: %w", posDestino/pageSize, err)
		}

		// Guardamos el nuevo marco (actualizado)
		nuevaPosiciones = append(nuevaPosiciones, posDestino/pageSize)
		posDestino += pageSize
	}

	// Truncar el archivo para eliminar los marcos vacíos al final
	err = swapFile.Truncate(int64(posDestino))
	if err != nil {
		return fmt.Errorf("error truncando swap.bin: %w", err)
	}

	// Actualizar la lista con las posiciones compactadas
	h.ProcesoPorPosicionSwap = nuevaPosiciones
	return nil
}

func (h *Handler) eliminarOcurrencias(pid int) {
	listaActualizada := make([]int, 0)
	for _, v := range h.ProcesoPorPosicionSwap {
		if v != pid {
			listaActualizada = append(listaActualizada, v)
		}
	}
	h.ProcesoPorPosicionSwap = listaActualizada
}

func (h *Handler) CargarPaginasEnMemoriaDesdeSwap(posicionesSwap []int, marcosDestino []int) error {
	if len(posicionesSwap) != len(marcosDestino) {
		return fmt.Errorf("la cantidad de posiciones y marcos no coincide")
	}

	archivoSwap, err := os.Open("/home/utnso/Desktop/tp-2025-1c-CPU-Warriors/memoria/swapfile.bin")
	if err != nil {
		panic(err)
	}
	//Para cerrarlo despues
	defer archivoSwap.Close()

	h.Log.Debug("CargarPaginasEnMemoriaDesdeSwap",
		log.AnyAttr("entre aca", posicionesSwap))

	for i, posSwap := range posicionesSwap {
		offset := int64(posSwap * h.Config.PageSize)
		buffer := make([]byte, h.Config.PageSize)

		_, err := archivoSwap.ReadAt(buffer, offset)
		if err != nil {

			h.Log.Debug("CargarPaginasEnMemoriaDesdeSwap",
				log.AnyAttr("err", err))
			h.Log.Error("Tamaño del Proceso no proporcionado")
		}

		// Copiar al EspacioDeUsuario en el marco correspondiente
		dest := marcosDestino[i] * h.Config.PageSize
		copy(h.EspacioDeUsuario[dest:dest+h.Config.PageSize], buffer)

		h.Log.Debug("CargarPaginasEnMemoriaDesdeSwap",
			log.AnyAttr("offset", offset))

		h.Log.Debug("CargarPaginasEnMemoriaDesdeSwap",
			log.AnyAttr("buffer", buffer))
	}

	return nil
}

func (h *Handler) PosicionesDeProcesoEnSwap(pid int) []int {
	posiciones := []int{}
	for i, p := range h.ProcesoPorPosicionSwap {
		if p == pid {
			posiciones = append(posiciones, i)
		}
	}
	return posiciones
}

func (h *Handler) DumpProceso(w http.ResponseWriter, r *http.Request) {

	var (
		//ctx = r.Context()
		// Leemos el PID
		pid = r.URL.Query().Get("pid")
	)

	if pid == "" {
		h.Log.Error("PID no proporcionado")
		http.Error(w, "PID no proporcionado", http.StatusBadRequest)
		return
	}
	/* HACER LOG

	“## PID: <PID> - Memory Dump solicitado”
	*/

	h.DumpProcesoFuncionAuxiliar(pid)

}

func (h *Handler) DumpProcesoFuncionAuxiliar(pid string) {

	h.Log.Debug("DumpFuncionAuxiliar",
		log.AnyAttr("pid", pid))

	procesYTablaAsociada, _ := h.BuscarProcesoPorPID(pid)
	h.Log.Debug("DumpProcesoFuncionAuxiliar",
		log.AnyAttr("procesYTablaAsociada", procesYTablaAsociada.TablasDePaginas))

	marcosDelProceso := h.ObtenerMarcosDeLaTabla(procesYTablaAsociada.TablasDePaginas)

	h.Log.Debug("DumpProcesoFuncionAuxiliar",
		log.AnyAttr("marcosDelProceso", marcosDelProceso))

	timestamp := time.Now().Format("20060102_150405")
	fileName := fmt.Sprintf("%s-%s.dmp", pid, timestamp)
	fullPath := filepath.Join(h.Config.DumpPath, fileName)

	// Crear el archivo
	file, err := os.Create(fullPath)
	if err != nil {
		h.Log.Error("Error creando el dump")
	}
	defer file.Close()

	pageSize := h.Config.PageSize

	for _, marco := range marcosDelProceso {
		offset := marco * pageSize
		pagina := h.EspacioDeUsuario[offset : offset+pageSize]

		_, err := file.Write(pagina)
		if err != nil {
			h.Log.Error("Error escribiendo el dump asociado al marco")
		}
	}

}

func (h *Handler) FinalizarProceso(w http.ResponseWriter, r *http.Request) {

	var (
		//ctx = r.Context()
		// Leemos el PID
		pid = r.URL.Query().Get("pid")
	)

	if pid == "" {
		h.Log.Error("PID no proporcionado")
		http.Error(w, "PID no proporcionado", http.StatusBadRequest)
		return
	}
	/* HACER LOG
	“## PID: <PID> - Proceso Destruido - Métricas - Acc.T.Pag: <ATP>;
	Inst.Sol.: <Inst.Sol.>;
	SWAP: <SWAP>;
	Mem.Prin.: <Mem.Prin.>;
	Lec.Mem.: <Lec.Mem.>;
	Esc.Mem.: <Esc.Mem.>”
	*/

	h.FinalizarProcesoFuncionAuxiliar(pid)

}

func (h *Handler) FinalizarProcesoFuncionAuxiliar(pid string) {

	pidInt, _ := strconv.Atoi(pid)

	if h.ContienePIDEnSwap(pidInt) {
		//compactar la posicion en swap y borrarlo en la lista de procesos}
		h.eliminarOcurrencias(pidInt)
		h.CompactarSwap()

	} else {
		procesYTablaAsociada, _ := h.BuscarProcesoPorPID(pid)
		h.Log.Debug("DumpProcesoFuncionAuxiliar",
			log.AnyAttr("procesYTablaAsociada", procesYTablaAsociada.TablasDePaginas))

		marcosDelProceso := h.ObtenerMarcosDeLaTabla(procesYTablaAsociada.TablasDePaginas)

		for marco := range marcosDelProceso {
			copy(h.EspacioDeUsuario[marco*h.Config.PageSize:((marco+1)*h.Config.PageSize-1)], make([]byte, h.Config.PageSize))
		}
	}
	//hasta aca el else
	//2do borrarlo de la lista de tablas
	h.BorrarProcesoPorPID(pid)
	//3ero borrar las instrucciones

	delete(h.Instrucciones, pidInt)

}

func (h *Handler) ContienePIDEnSwap(pid int) bool {
	for _, valor := range h.ProcesoPorPosicionSwap {
		if valor == pid {
			return true
		}
	}
	return false
}

func (h *Handler) BorrarProcesoPorPID(pid string) error {
	for i, proceso := range h.TablasProcesos {
		if proceso.PID == pid {
			// Borramos el elemento del slice
			h.TablasProcesos = append(h.TablasProcesos[:i], h.TablasProcesos[i+1:]...)
			return nil
		}
	}
	return fmt.Errorf("proceso con PID %s no encontrado", pid)
}

func (h *Handler) LeerPaginaCompleta(marco int, pid string) {
	h.LeerPagina(marco, 0, h.Config.PageSize, pid)
}

func (h *Handler) ActualizarPaginaCompleta(marco int, valorAEscribir string, pid string) {
	h.EscribirPagina(marco, 0, valorAEscribir, pid)
}

func (h *Handler) LeerPagina(marco int, offset int, tamanioALeer int, pid string) string {

	//if tamanioALeer mayor a cero
	lecturaMemoria := string(h.EspacioDeUsuario[((marco * h.Config.PageSize) + offset):((marco * h.Config.PageSize) + offset + tamanioALeer + 1)])

	h.Log.Debug("LeerPagina",
		log.AnyAttr("lecturaMemoria", lecturaMemoria))
	return lecturaMemoria

	/* HACER LOG
	“## PID: <PID> - <Lectura> - Dir. Física: <DIRECCIÓN_FÍSICA> - Tamaño: <TAMAÑO>”*/
}

func (h *Handler) EscribirPagina(marco int, offset int, valorAEscribir string, pid string) {

	copy(h.EspacioDeUsuario[((marco*h.Config.PageSize)+offset):], []byte(valorAEscribir))

	h.Log.Debug("EscribirPagina",
		log.AnyAttr("lecturaMemoria", h.EspacioDeUsuario))

	/* HACER LOG
	“## PID: <PID> - <Escritura> - Dir. Física: <DIRECCIÓN_FÍSICA> - Tamaño: <TAMAÑO>”*/

}

// [0 , 0 , 1]
func (h *Handler) BuscarMarcoPorPagina(paginas []int, pid string) int {

	procesYTablaAsociada, _ := h.BuscarProcesoPorPID(pid)

	tablasDePAginas := procesYTablaAsociada.TablasDePaginas
	marco, _ := h.ObtenerMarcoDesdeIndices(tablasDePAginas, paginas)

	h.Log.Debug("BuscarMarcoPorPagina",
		log.AnyAttr("marco", marco))
	return 1
}

func (h *Handler) ObtenerMarcoDesdeIndices(tablas []interface{}, indices []int) (int, error) {
	actual := tablas[0].([]*TablaIntermedia)

	siguienteNivelIdx := 0

	for _, idx := range indices {

		h.Log.Debug("tablaIntermedia",
			log.AnyAttr("forEntre", "marco"))

		// Accedés a una tabla específica, por ejemplo la primera o la que te interese por índice:
		tabla := actual[siguienteNivelIdx] // ← Acá accedés a una *TablaIntermedia

		// Ahora sí podés acceder a sus entradas:

		//[{"Entradas":[0,1]}]
		siguienteNivelIdx, _ = tabla.Entradas[siguienteNivelIdx].(int)

		// Bajamos un nivel
		actual = tablas[idx+1].([]*TablaIntermedia)

		h.Log.Debug("ObtenerMarcoDesdeIndices",
			log.AnyAttr("ACTUAL !!!!!!!!!", actual))

		///actual = siguienteNivel
	}

	return -1, fmt.Errorf("no se pudo encontrar el marco")
}

func (h *Handler) CrearTabla(niveles int, entradasPorElemento int) interface{} {
	if niveles <= 0 {
		return nil
	}

	if niveles == 1 {
		hoja := make([]int, entradasPorElemento)
		for i := range hoja {
			hoja[i] = -1
		}
		return hoja
	}

	tabla := make([]interface{}, entradasPorElemento)
	for i := 0; i < entradasPorElemento; i++ {
		tabla[i] = h.CrearTabla(niveles-1, entradasPorElemento)
	}

	h.Log.Debug("CrearTabla",
		log.AnyAttr("myVar", tabla))

	return tabla
}

func (h *Handler) LeerValor(tabla interface{}, indices []int) (int, bool) {
	actual := tabla
	for i := 0; i < len(indices); i++ {
		switch nodo := actual.(type) {
		case []interface{}:
			if indices[i] < 0 || indices[i] >= len(nodo) {
				return 0, false
			}
			actual = nodo[indices[i]]
		case []int:
			if indices[i] < 0 || indices[i] >= len(nodo) {
				return 0, false
			}
			return nodo[indices[i]], true
		default:
			return 0, false
		}
	}
	return 0, false
}

func (h *Handler) LlenarTablaConValores(tabla interface{}, valores []int) {
	var index int

	var recorrer func(nodo interface{}) interface{}

	recorrer = func(nodo interface{}) interface{} {
		switch t := nodo.(type) {
		case []interface{}:
			for i := range t {
				t[i] = recorrer(t[i])
			}
			return t
		case []int:
			for i := range t {
				if index < len(valores) {
					t[i] = valores[index]
					index++
				} // si no hay más valores, se queda en -1
			}
			return t
		default:
			return t
		}
	}

	recorrer(tabla)
}
