package api

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
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

type LecturaEscrituraBody struct {
	PID            string `json:"pid"`
	Frame          int    `json:"frame"`
	Offset         int    `json:"offset"`
	Tamanio        int    `json:"tamanio"`
	ValorAEscribir string `json:"valor_a_escribir,omitempty"`
}

type CacheData struct {
	PID              string `json:"pid"`
	Data             string `json:"data"`
	EntradasPorNivel string `json:"entradas_por_nivel"`
}

// TablasProceso Estructura en la cual guardamos las metricas y la tabla de paginas
type TablasProceso struct {
	PID             string      `json:"pid"`
	Tamanio         int         `json:"tamanio_proceso"`
	TablasDePaginas interface{} `json:"tabla_de_paginas"`

	CantidadAccesosATablas           int `json:"cantidad_accesos_a_tablas"`
	CantidadInstruccionesSolicitadas int `json:"cantidad_instrucciones_solicitadas"`
	CantidadBajadasSwap              int `json:"cantidad_bajadas_swap"`
	CantidadSubidasMemoriaPrincipal  int `json:"cantidad_subidas_memoria_principal"`
	CantidadDeEscritura              int `json:"cantidad_de_escritura"`
	CantidadDeLectura                int `json:"cantidad_de_lectura"`
}

// ConsultarEspacioEInicializar recibe una consulta sobre el espacio libre en memoria. (Lo recibe cuando el proceso pasa al ready ya que ahi empeiza a ocupar memoria)
// En caso de que haya espacio, se inicializa el proceso, se responde con un mensaje de éxito y el tamaño disponible.
// En caso contrario, se responde con un mensaje de error.
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

	if 0 <= paginasLibres-paginasNecesarias {
		h.AsignarMemoriaDeUsuario(paginasNecesarias, pid)
	} else {
		h.Log.Error("No hay espacio disponible",
			log.IntAttr("PaginasLibres", paginasLibres),
			log.IntAttr("PaginasNecesarias", paginasNecesarias))
		http.Error(w, "no hay espacio disponible", http.StatusInsufficientStorage)
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

// DivRedondeoArriba Redondea para arriba al dividir el (tamanio de proceso / paginas), esto es xq no le puedo
// asignar una página y media o es 1 o son 2
func DivRedondeoArriba(numerador, denominador int) int {
	return (numerador + denominador - 1) / denominador
}

// ContarLibres Cuenta cuantas páginas libres hay y las devuelve en un int la cantidad
func (h *Handler) ContarLibres() int {
	libres := 0
	for _, ocupado := range h.FrameTable {
		if !ocupado {
			libres++
		}
	}
	return libres
}

// AsignarMemoriaDeUsuario Si hay espacio en ConsultarEspacioEInicializar entonces entra aca
// Verifica si el proceso existe (Salde suspBLocked es decir de Swap) crea solo la tabla de páginas
// Si el proceso no exisita en memoria le crea un TablasProceso, donde además le genere la tabla de páginas
// vacia y le asigna los frames
func (h *Handler) AsignarMemoriaDeUsuario(paginasAOcupar int, pid string) {
	if paginasAOcupar != 0 {
		framesLibres := h.MarcosLibres(paginasAOcupar)

		tabla := h.CrearTabla(h.Config.NumberOfLevels, h.Config.EntriesPerPage)

		h.LlenarTablaConValores(tabla, framesLibres)

		// Actualizar el espacio de usuario con los marcos ocupados
		for _, marco := range framesLibres {
			h.FrameTable[marco] = true // Marcar el marco como ocupado
			h.Log.Debug("AsignarMemoriaDeUsuario",
				log.IntAttr("marco", marco),
				log.StringAttr("pid", pid))
		}

		h.Log.Debug("AsignarMemoriaDeUsuario",
			log.AnyAttr("tabla", tabla))

		pidInt, _ := strconv.Atoi(pid)
		if h.ContienePIDEnSwap(pidInt) {
			for _, tp := range h.TablasProcesos {
				if tp.PID == pid {
					tp.TablasDePaginas = tabla
				}
			}
			h.SacarProcesoDeSwap(pid)
		} else {
			tablaProceso := &TablasProceso{
				PID:                              pid,
				Tamanio:                          paginasAOcupar * h.Config.PageSize,
				TablasDePaginas:                  tabla,
				CantidadAccesosATablas:           0,
				CantidadInstruccionesSolicitadas: 0,
				CantidadBajadasSwap:              0,
				CantidadSubidasMemoriaPrincipal:  0,
				CantidadDeEscritura:              0,
				CantidadDeLectura:                0,
			}
			h.TablasProcesos = append(h.TablasProcesos, tablaProceso)

			//Log obligatorio: Creación de Proceso
			//  “## PID: <PID> - Proceso Creado - Tamaño: <TAMAÑO>”
			h.Log.Info(fmt.Sprintf("“## PID: %s - Proceso Creado - Tamaño: %d", pid, paginasAOcupar*h.Config.PageSize))
		}
	} else {
		tablaProceso := &TablasProceso{
			PID:                              pid,
			Tamanio:                          paginasAOcupar * h.Config.PageSize,
			CantidadAccesosATablas:           0,
			CantidadInstruccionesSolicitadas: 0,
			CantidadBajadasSwap:              0,
			CantidadSubidasMemoriaPrincipal:  0,
			CantidadDeEscritura:              0,
			CantidadDeLectura:                0,
		}
		h.TablasProcesos = append(h.TablasProcesos, tablaProceso)

		//Log obligatorio: Creación de Proceso
		//  “## PID: <PID> - Proceso Creado - Tamaño: <TAMAÑO>”
		h.Log.Info(fmt.Sprintf("“## PID: %s - Proceso Creado - Tamaño: %d", pid, paginasAOcupar*h.Config.PageSize))
	}

	tablaMetricas, _ := h.BuscarProcesoPorPID(pid)
	tablaMetricas.CantidadSubidasMemoriaPrincipal++

}

// Recibe el archivo Swap y el marco que debe pasar a Swap y lo escribe en el swap
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

// BuscarProcesoPorPID Busca la tablaProceso dentro de la lista de TablasProcesos por pid
func (h *Handler) BuscarProcesoPorPID(pid string) (*TablasProceso, error) {
	for _, proceso := range h.TablasProcesos {
		if proceso.PID == pid {
			return proceso, nil
		}
	}
	return nil, fmt.Errorf("proceso con PID %s no encontrado", pid)
}

// MarcosLibres Le pasamos cuantas paginas necesitamos y nos devuelve una lista de cuales debemos usar
func (h *Handler) MarcosLibres(paginasNecesarias int) []int {
	libres := make([]int, 0)
	for i, ocupado := range h.FrameTable {
		if !ocupado { // Si el marco está libre, su valor es false
			libres = append(libres, i)
			if len(libres) == paginasNecesarias {
				return libres
			}
		}
	}
	return libres
}

// CargarProcesoEnMemoriaDeSistema Carga las instrucciones en la memoria del sistema, tener en cuenta que esta no es limitada en cuanto a espacio
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

	h.mutexInstrucciones.Lock()
	if h.Instrucciones[pidInt] == nil {
		h.Instrucciones[pidInt] = make([]Instruccion, 0)
	}
	h.mutexInstrucciones.Unlock()

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

		h.mutexInstrucciones.Lock()
		h.Instrucciones[pidInt] = append(h.Instrucciones[pidInt], instruccion)
		h.mutexInstrucciones.Unlock()
	}

	h.Log.Debug("Carga de Proceso en Memoria de Sistema Exitosa",
		log.StringAttr("pid", pid),
		log.StringAttr("file_path", filePath),
	)

	w.WriteHeader(http.StatusOK)
}

// PasarProcesoASwap Recibe la llamada del Kernel cuando un proceso se suspendio, y lo pasa  Swap usando PasarProcesoASwapAuxiliar
func (h *Handler) PasarProcesoASwap(w http.ResponseWriter, r *http.Request) {
	var (
		// Leemos el PID
		pid = r.URL.Query().Get("pid")
	)

	if pid == "" {
		h.Log.Error("PID no proporcionado")
		http.Error(w, "PID no proporcionado", http.StatusBadRequest)
		return
	}

	time.Sleep(time.Duration(h.Config.SwapDelay) * time.Millisecond)
	h.PasarProcesoASwapAuxiliar(pid)

	// Devolvemos una respuesta exitosa
	w.WriteHeader(http.StatusOK)
}

// PasarProcesoASwapAuxiliar Recibe el PID, con eso utiliza BuscarProcesoPorPID para traer la tablaProceso
// Con ObtenerMarcosDeLaTabla obtiene los marcos, libera el bitmap de memoriaDeUsuario
// escribirMarcoEnSwap escribe efectivamente en swap los marcos que le pasamos
// por último actualizamos las metricas
func (h *Handler) PasarProcesoASwapAuxiliar(pid string) {
	procesYTablaAsociada, _ := h.BuscarProcesoPorPID(pid)
	h.Log.Debug("PasarProcesoASwapAuxiliar",
		log.AnyAttr("procesYTablaAsociada", procesYTablaAsociada.TablasDePaginas))

	marcosDelProceso := h.ObtenerMarcosDeLaTabla(procesYTablaAsociada.TablasDePaginas)

	// Actualizar el espacio de usuario con los marcos libres
	for _, marco := range marcosDelProceso {
		h.FrameTable[marco] = false // Marcar el marco como libre
	}

	h.Log.Debug("PasarProcesoASwapAuxiliar",
		log.AnyAttr("ObtenerMarcosValidos", marcosDelProceso))

	//iterar la lista de marcos un for, y por cada uno multiplicarlo por el sizepage

	archivoSwap, err := os.OpenFile(h.Config.SwapfilePath, os.O_CREATE|os.O_APPEND|os.O_RDWR, 0644)
	if err != nil {
		panic(err)
	}
	//Para cerrarlo despues
	defer func(f *os.File) {
		_ = f.Close()
	}(archivoSwap)

	pidInt, _ := strconv.Atoi(pid)

	for i := 0; i < len(marcosDelProceso); i++ {
		h.ProcesoPorPosicionSwap = append(h.ProcesoPorPosicionSwap, pidInt)
	}

	for _, marco := range marcosDelProceso {
		err = h.escribirMarcoEnSwap(archivoSwap, marco)
		if err != nil {
			panic(err)
		}

		copy(h.EspacioDeUsuario[marco*h.Config.PageSize:(marco+1)*h.Config.PageSize], make([]byte, h.Config.PageSize))
	}
	tablaMetricas, _ := h.BuscarProcesoPorPID(pid)
	tablaMetricas.CantidadBajadasSwap++
}

// ObtenerMarcosDeLaTabla Le pasamos la tabla de paginas y nos devuelve los marcos ocupados en esa tabla
func (h *Handler) ObtenerMarcosDeLaTabla(tabla interface{}) []int {
	var marcos []int

	switch t := tabla.(type) {
	case []int:
		// Caso hoja: extraer los marcos válidos
		for _, marco := range t {
			if marco != -1 {
				marcos = append(marcos, marco)
			}
		}
	case []interface{}:
		// Caso intermedio: recorrer cada subnivel
		for _, sub := range t {
			marcos = append(marcos, h.ObtenerMarcosDeLaTabla(sub)...)
		}
	default:
		// Tipo no reconocido (no debería ocurrir)
	}

	return marcos
}

// SacarProcesoDeSwap Es llamada desde AsignarMemoriaDeUsuario si el proceso sale del bloque a ready
// se encarga de buscar en swap ese proceso y actualizar la memoria de usuario
func (h *Handler) SacarProcesoDeSwap(pid string) {
	pidDeSwap, _ := strconv.Atoi(pid)

	posicionEnSwap := h.PosicionesDeProcesoEnSwap(pidDeSwap)

	procesYTablaAsociadaDeSwap, _ := h.BuscarProcesoPorPID(pid)
	marcosDelProcesoDeSwap := h.ObtenerMarcosDeLaTabla(procesYTablaAsociadaDeSwap.TablasDePaginas)

	//ir escribiendo cada frame en memoria
	if err := h.CargarPaginasEnMemoriaDesdeSwap(posicionEnSwap, marcosDelProcesoDeSwap); err != nil {
		h.Log.Error("Error al cargar páginas en memoria desde swap",
			log.ErrAttr(err),
			log.AnyAttr("posicionEnSwap", posicionEnSwap),
			log.AnyAttr("marcosDelProcesoDeSwap", marcosDelProcesoDeSwap),
		)
		return
	}
	h.Log.Debug("SacarProcesoDeSwap",
		log.AnyAttr("CargarPaginasEnMemoriaDesdeSwap", h.EspacioDeUsuario))

	// Compactar el swap eliminando el contenido del proceso
	if err := h.CompactarSwap(pid); err != nil {
		h.Log.Error("Error al compactar swap",
			log.ErrAttr(err),
		)
		return
	}
	time.Sleep(time.Duration(h.Config.SwapDelay) * time.Millisecond)
}

// CompactarSwap Se encarga de compactar tanto la swap como elk bitmap de la swap
func (h *Handler) CompactarSwap(pidAEliminar string) error {
	pageSize := h.Config.PageSize
	pidInt, _ := strconv.Atoi(pidAEliminar)

	// Abrimos el archivo en modo lectura/escritura
	swapFile, err := os.OpenFile(h.Config.SwapfilePath, os.O_RDWR, 0644)
	if err != nil {
		return fmt.Errorf("error abriendo swap.bin: %w", err)
	}
	defer func(f *os.File) {
		_ = f.Close()
	}(swapFile)

	posDestino := 0
	nuevaPosiciones := make([]int, 0)

	// Recorremos la lista actual y solo mantenemos los que NO son del PID a eliminar
	for i, pid := range h.ProcesoPorPosicionSwap {
		if pid != pidInt {
			// Este marco NO pertenece al PID a eliminar, lo conservamos
			offset := i * pageSize
			buffer := make([]byte, pageSize)

			// Leer el marco original
			_, err := swapFile.ReadAt(buffer, int64(offset))
			if err != nil && err != io.EOF {
				return fmt.Errorf("error leyendo marco %d: %w", i, err)
			}

			// Escribirlo en la nueva posición compactada
			_, err = swapFile.WriteAt(buffer, int64(posDestino))
			if err != nil {
				return fmt.Errorf("error escribiendo marco en pos %d: %w", posDestino/pageSize, err)
			}

			// Guardamos la nueva posición y el PID correspondiente
			nuevaPosiciones = append(nuevaPosiciones, pid)
			posDestino += pageSize
		}
	}

	// Truncar el archivo para eliminar los marcos no utilizados al final
	err = swapFile.Truncate(int64(posDestino))
	if err != nil {
		return fmt.Errorf("error truncando swap.bin: %w", err)
	}

	// Actualizar la lista con las posiciones compactadas (sin el PID eliminado)
	h.ProcesoPorPosicionSwap = nuevaPosiciones

	return nil
}

// CargarPaginasEnMemoriaDesdeSwap Carga las paginas desde Swap a memmoria de usuario
func (h *Handler) CargarPaginasEnMemoriaDesdeSwap(posicionesSwap []int, marcosDestino []int) error {
	if len(posicionesSwap) != len(marcosDestino) {
		return fmt.Errorf("la cantidad de posiciones y marcos no coincide")
	}

	archivoSwap, err := os.Open(h.Config.SwapfilePath)
	if err != nil {
		panic(err)
	}
	//Para cerrarlo despues
	defer func(f *os.File) {
		_ = f.Close()
	}(archivoSwap)

	h.Log.Debug("CargarPaginasEnMemoriaDesdeSwap",
		log.AnyAttr("entre aca", posicionesSwap))

	for i, posSwap := range posicionesSwap {
		offset := int64(posSwap * h.Config.PageSize)
		buffer := make([]byte, h.Config.PageSize)

		_, err := archivoSwap.ReadAt(buffer, offset)
		if err != nil {
			h.Log.Error("Tamaño del Proceso no proporcionado",
				log.AnyAttr("err", err))
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

// PosicionesDeProcesoEnSwap Busca en el "Bitmap" del Swap y devuelve en que posciones del swap esta
func (h *Handler) PosicionesDeProcesoEnSwap(pid int) []int {
	posiciones := make([]int, 0)
	for i, p := range h.ProcesoPorPosicionSwap {
		if p == pid {
			posiciones = append(posiciones, i)
		}
	}
	return posiciones
}

// DumpProceso Recibe la llamada del Kernel para hacer el dump, usa DumpProcesoFuncionAuxiliar
func (h *Handler) DumpProceso(w http.ResponseWriter, r *http.Request) {
	var (
		// Leemos el PID
		pid = r.URL.Query().Get("pid")
	)

	if pid == "" {
		h.Log.Error("PID no proporcionado")
		http.Error(w, "PID no proporcionado", http.StatusBadRequest)
		return
	}
	/*Log obligatorio: Memory Dump
	"## PID: <PID> - Memory Dump solicitado"
	*/
	h.Log.Info(fmt.Sprintf("## PID: %s - Memory Dump solicitado", pid))

	if err := h.DumpProcesoFuncionAuxiliar(pid); err != nil {
		h.Log.Error("Error al crear el dump del proceso",
			log.ErrAttr(err),
			log.StringAttr("pid", pid),
		)
		http.Error(w, "Error al crear el dump del proceso", http.StatusInternalServerError)
		return
	}

	// Enviamos una respuesta exitosa
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("Dump del proceso creado exitosamente"))
}

// DumpProcesoFuncionAuxiliar Busca con el pid la tabla asociada para luego hacer el dump
func (h *Handler) DumpProcesoFuncionAuxiliar(pid string) error {
	h.Log.Debug("DumpFuncionAuxiliar",
		log.AnyAttr("pid", pid))

	// Verificar que el proceso existe
	procesYTablaAsociada, err := h.BuscarProcesoPorPID(pid)
	if err != nil {
		h.Log.Error("Error al buscar proceso por PID",
			log.StringAttr("pid", pid),
			log.ErrAttr(err))
		return fmt.Errorf("proceso no encontrado: %w", err)
	}

	h.Log.Debug("DumpProcesoFuncionAuxiliar",
		log.AnyAttr("procesYTablaAsociada", procesYTablaAsociada.TablasDePaginas))

	// Verificar que la tabla de páginas no es nil
	if procesYTablaAsociada.TablasDePaginas == nil {
		h.Log.Error("Tabla de páginas es nil",
			log.StringAttr("pid", pid))
		return fmt.Errorf("tabla de páginas es nil para proceso %s", pid)
	}

	marcosDelProceso := h.ObtenerMarcosDeLaTabla(procesYTablaAsociada.TablasDePaginas)

	h.Log.Debug("DumpProcesoFuncionAuxiliar",
		log.AnyAttr("marcosDelProceso", marcosDelProceso))

	// Verificar que hay marcos válidos
	if len(marcosDelProceso) == 0 {
		h.Log.Error("No hay marcos válidos para el proceso",
			log.StringAttr("pid", pid))
		return fmt.Errorf("no hay marcos válidos para proceso %s", pid)
	}

	// Crear el directorio si no existe
	if err := os.MkdirAll(h.Config.DumpPath, 0755); err != nil {
		h.Log.Error("Error creando directorio de dump",
			log.StringAttr("dump_path", h.Config.DumpPath),
			log.ErrAttr(err))
		return fmt.Errorf("error creando directorio de dump: %w", err)
	}

	timestamp := time.Now().Format("20060102_150405")
	fileName := fmt.Sprintf("%s-%s.dmp", pid, timestamp)
	fullPath := filepath.Join(h.Config.DumpPath, fileName)

	// Crear el archivo
	file, err := os.Create(fullPath)
	if err != nil {
		h.Log.Error("Error creando archivo de dump",
			log.StringAttr("fullPath", fullPath),
			log.ErrAttr(err))
		return fmt.Errorf("error creando archivo de dump: %w", err)
	}

	defer func(f *os.File) {
		if err := f.Close(); err != nil {
			h.Log.Error("Error cerrando archivo de dump",
				log.StringAttr("fullPath", fullPath),
				log.ErrAttr(err))
		}
	}(file)

	pageSize := h.Config.PageSize

	for _, marco := range marcosDelProceso {
		// Verificar que el marco es válido
		if marco < 0 || marco >= len(h.EspacioDeUsuario)/pageSize {
			h.Log.Error("Marco inválido detectado",
				log.IntAttr("marco", marco),
				log.IntAttr("max_marcos", len(h.EspacioDeUsuario)/pageSize))
			continue
		}

		offset := marco * pageSize
		// Verificar que no excedemos el límite del espacio de usuario
		if offset+pageSize > len(h.EspacioDeUsuario) {
			h.Log.Error("Offset excede el espacio de usuario",
				log.IntAttr("offset", offset),
				log.IntAttr("pageSize", pageSize),
				log.IntAttr("espacioDeUsuario", len(h.EspacioDeUsuario)))
			continue
		}

		pagina := h.EspacioDeUsuario[offset : offset+pageSize]

		_, err = file.Write(pagina)
		if err != nil {
			h.Log.Error("Error escribiendo el dump asociado al marco",
				log.IntAttr("marco", marco),
				log.ErrAttr(err))
			return fmt.Errorf("error escribiendo el dump asociado al marco %d: %w", marco, err)
		}
	}

	return nil
}

// ContienePIDEnSwap Busca si el PID se encuentra en el bitmap de swap
func (h *Handler) ContienePIDEnSwap(pid int) bool {
	for _, valor := range h.ProcesoPorPosicionSwap {
		if valor == pid {
			return true
		}
	}
	return false
}

// ActualizarPaginaCompleta Recibe la llamada de CPU para realizar la actualizacion de una página que se encontraba en caché
func (h *Handler) ActualizarPaginaCompleta(w http.ResponseWriter, r *http.Request) {
	var (
		body             CacheData // La key es el índice de la página, el valor es un CacheData con PID y Data
		cantAccesosTabla int
	)

	// Leo el cuerpo de la solicitud y guardo el valor del body en la variable
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		h.Log.Error("Error al decodificar interrupción",
			log.ErrAttr(err))
		http.Error(w, "error al decodificar mensaje", http.StatusInternalServerError)
		return
	}

	pid := body.PID
	data := body.Data

	var indices = make([]int, 0)
	indicesSplit := strings.Split(body.EntradasPorNivel, "-")
	for _, indice := range indicesSplit {
		paginaInt, err := strconv.Atoi(indice)
		if err != nil {
			h.Log.Error("Error al convertir entrada de nivel a entero",
				log.ErrAttr(err),
				log.StringAttr("entrada", indice),
			)
			http.Error(w, "error al convertir entrada de nivel a entero", http.StatusBadRequest)
			return
		}
		indices = append(indices, paginaInt)
	}

	tablaMetricas, err := h.BuscarProcesoPorPID(pid)
	if err != nil {
		h.Log.Error("Error al buscar proceso por PID",
			log.StringAttr("pid", pid),
			log.ErrAttr(err))
		http.Error(w, "proceso no encontrado", http.StatusNotFound)
		return
	}
	tablaMetricas.CantidadDeEscritura++

	cantAccesosTabla += len(indices)
	frame, found := buscarMarcoPorPaginaAux(tablaMetricas, indices)
	if !found {
		h.Log.Error("Marco no encontrado para la página",
			log.AnyAttr("indices", indices),
			log.StringAttr("pid", pid))
		http.Error(w, "marco no encontrado para la página", http.StatusNotFound)
		return
	}
	copy(h.EspacioDeUsuario[frame*h.Config.PageSize:(frame+1)*h.Config.PageSize], data)

	/* Log obligatorio: Escritura / lectura en espacio de usuario
	"## PID: <PID> - <Escritura> - Dir. Física: <DIRECCIÓN_FÍSICA> - Tamaño: <TAMAÑO>"*/
	h.Log.Info(fmt.Sprintf("## PID: %s - ESCRITURA %s - Dir. Física: %d - Tamaño: %d",
		pid, data, frame*h.Config.PageSize, len(data)))

	h.Log.Debug(fmt.Sprintf("## PID: %s - Actualización de página completa - Dir. Física: %d - Tamaño: %d",
		pid, frame*h.Config.PageSize, len(data)))

	// Aplicamos el retardo de memoria configurado * cantidad de accesos a tablas
	time.Sleep(time.Duration(h.Config.MemoryDelay*cantAccesosTabla) * time.Millisecond)

	// Enviamos una respuesta exitosa
	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write([]byte(`{Ok}`))
}

// LeerPagina Recibe la instruccion READ de CPU
func (h *Handler) LeerPagina(w http.ResponseWriter, r *http.Request) {
	var (
		ctx     = r.Context()
		lectura = LecturaEscrituraBody{}
	)
	// Leo el cuerpo de la solicitud y guardo el valor del body en la variable
	if err := json.NewDecoder(r.Body).Decode(&lectura); err != nil {
		h.Log.ErrorContext(ctx, "Error al decodificar interrupción",
			log.ErrAttr(err))
		http.Error(w, "error al decodificar mensaje", http.StatusInternalServerError)
		return
	}

	dl := lectura.Frame*h.Config.PageSize + lectura.Offset
	//if tamanioALeer mayor a cero
	lecturaMemoria := string(h.EspacioDeUsuario[dl:(dl + lectura.Tamanio)])

	lecturaMemoria = h.limpiarNulos(lecturaMemoria)
	/* Log obligatorio: Escritura / lectura en espacio de usuario
	"## PID: <PID> - <Lectura> - Dir. Física: <DIRECCIÓN_FÍSICA> - Tamaño: <TAMAÑO>"*/
	h.Log.Info(fmt.Sprintf("## PID: %s - %s - Dir. Física: %d - Tamaño: %d",
		lectura.PID, lecturaMemoria, lectura.Frame*h.Config.PageSize+lectura.Offset, lectura.Tamanio))

	tablaMetricas, _ := h.BuscarProcesoPorPID(lectura.PID)
	tablaMetricas.CantidadDeLectura++

	h.Log.Debug("LeerPagina",
		log.AnyAttr("lecturaMemoria", lecturaMemoria))

	time.Sleep(time.Duration(h.Config.MemoryDelay) * time.Millisecond)

	// Enviamos la respuesta al cliente con el contenido leído
	w.Header().Set("Content-Type", "application/json")
	response := map[string]string{
		"contenido": lecturaMemoria,
	}

	responseBody, _ := json.Marshal(response)
	// Agrego el status Code 200 a la respuesta
	w.WriteHeader(http.StatusOK)
	// Envío la respuesta al cliente
	_, _ = w.Write(responseBody)
}

// LeerPaginaCompleta Recibe la instruccion READ desde Cache
func (h *Handler) LeerPaginaCompleta(w http.ResponseWriter, r *http.Request) {
	var (
		ctx     = r.Context()
		lectura = LecturaEscrituraBody{}
	)
	// Leo el cuerpo de la solicitud y guardo el valor del body en la variable
	if err := json.NewDecoder(r.Body).Decode(&lectura); err != nil {
		h.Log.ErrorContext(ctx, "Error al decodificar interrupción",
			log.ErrAttr(err))
		http.Error(w, "error al decodificar mensaje", http.StatusInternalServerError)
		return
	}

	dl := lectura.Frame * h.Config.PageSize
	//if tamanioALeer mayor a cero
	lecturaMemoria := string(h.EspacioDeUsuario[dl:(dl + h.Config.PageSize)])

	lecturaMemoria = h.limpiarNulos(lecturaMemoria)
	/* Log obligatorio: Escritura / lectura en espacio de usuario
	"## PID: <PID> - <Lectura> - Dir. Física: <DIRECCIÓN_FÍSICA> - Tamaño: <TAMAÑO>"*/
	h.Log.Info(fmt.Sprintf("## PID: %s - %s - Dir. Física: %d - Tamaño: %d",
		lectura.PID, lecturaMemoria, lectura.Frame*h.Config.PageSize, h.Config.PageSize))

	tablaMetricas, _ := h.BuscarProcesoPorPID(lectura.PID)
	tablaMetricas.CantidadDeLectura++

	h.Log.Debug("LeerPagina",
		log.AnyAttr("lecturaMemoria", lecturaMemoria))

	time.Sleep(time.Duration(h.Config.MemoryDelay) * time.Millisecond)

	// Enviamos la respuesta al cliente con el contenido leído
	w.Header().Set("Content-Type", "application/json")
	response := map[string]string{
		"contenido": lecturaMemoria,
	}

	responseBody, _ := json.Marshal(response)
	// Agrego el status Code 200 a la respuesta
	w.WriteHeader(http.StatusOK)
	// Envío la respuesta al cliente
	_, _ = w.Write(responseBody)
}

// limpiarNulos Limpia los espacios nules para hacer la lectura más legible
func (h *Handler) limpiarNulos(cadena string) string {
	return strings.ReplaceAll(cadena, "\x00", "")
}

// EscribirPagina Recibe la instruccion WRITE de CPU
func (h *Handler) EscribirPagina(w http.ResponseWriter, r *http.Request) {
	var (
		ctx       = r.Context()
		escritura = LecturaEscrituraBody{}
	)
	// Leo el cuerpo de la solicitud y guardo el valor del body en la variable interrupcion
	if err := json.NewDecoder(r.Body).Decode(&escritura); err != nil {
		h.Log.ErrorContext(ctx, "Error al decodificar interrupción",
			log.ErrAttr(err))
		http.Error(w, "error al decodificar mensaje", http.StatusInternalServerError)
		return
	}

	dl := escritura.Frame*h.Config.PageSize + escritura.Offset

	if dl < 0 || dl >= len(h.EspacioDeUsuario) {
		h.Log.ErrorContext(ctx, "Error de escritura fuera de límites",
			log.IntAttr("direccion_logica", dl),
			log.IntAttr("tamanio_espacio_usuario", len(h.EspacioDeUsuario)))
		http.Error(w, "error de escritura fuera de límites", http.StatusBadRequest)
		return
	}

	copy(h.EspacioDeUsuario[dl:dl+len(escritura.ValorAEscribir)], escritura.ValorAEscribir)

	h.Log.Debug("EscribirPagina",
		log.AnyAttr("lecturaMemoria", h.EspacioDeUsuario))

	tablaMetricas, _ := h.BuscarProcesoPorPID(escritura.PID)
	tablaMetricas.CantidadDeEscritura++
	/* Log obligatorio: Escritura / lectura en espacio de usuario
	"## PID: <PID> - <Escritura> - Dir. Física: <DIRECCIÓN_FÍSICA> - Tamaño: <TAMAÑO>"*/
	h.Log.Info(fmt.Sprintf("## PID: %s - ESCRITURA %s - Dir. Física: %d - Tamaño: %d",
		escritura.PID, escritura.ValorAEscribir, escritura.Frame*h.Config.PageSize+escritura.Offset, len(escritura.ValorAEscribir)))

	time.Sleep(time.Duration(h.Config.MemoryDelay) * time.Millisecond)

	// Devolvemos un status 200 OK
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("OK"))
}

// CrearTabla funcion que se encarga de crear la tabla de paginas
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

// BuscarMarcoPorPagina busca el marco correspondiente a una página dada en la tabla de páginas del proceso.
// Recibe el PID del proceso y la página a buscar como parámetros.
func (h *Handler) BuscarMarcoPorPagina(w http.ResponseWriter, r *http.Request) {
	var (
		// Leemos el PID y la página/dirección lógica de la consulta
		pid           = r.URL.Query().Get("pid")
		entradasNivel = r.URL.Query().Get("entradas-nivel")
	)

	if pid == "" {
		h.Log.Error("PID no proporcionado")
		http.Error(w, "PID no proporcionado", http.StatusBadRequest)
		return
	}

	var err error

	// Determinar si se proporcionó página o dirección lógica
	if entradasNivel == "" {
		h.Log.Error("Entradas por nivel faltantes",
			log.StringAttr("paginas", entradasNivel))
		http.Error(w, "entradas por nivel faltantes", http.StatusBadRequest)
		return
	}

	tablaProceso, err := h.BuscarProcesoPorPID(pid)
	if err != nil {
		h.Log.Error("Error al buscar proceso por PID",
			log.StringAttr("pid", pid),
			log.ErrAttr(err))
		http.Error(w, "proceso no encontrado", http.StatusNotFound)
		return
	}

	h.Log.Debug("BuscarMarcoPorPagina - índices calculados",
		log.StringAttr("pid", pid),
	)

	var indices = make([]int, 0)
	indicesSplit := strings.Split(entradasNivel, "-")
	for _, indice := range indicesSplit {
		paginaInt, err := strconv.Atoi(indice)
		if err != nil {
			h.Log.Error("Error al convertir entrada de nivel a entero",
				log.ErrAttr(err),
				log.StringAttr("entrada", indice),
			)
			http.Error(w, "error al convertir entrada de nivel a entero", http.StatusBadRequest)
			return
		}
		indices = append(indices, paginaInt)
	}

	frame, found := buscarMarcoPorPaginaAux(tablaProceso, indices)
	if !found {
		h.Log.Error("Error al buscar marco por página",
			log.StringAttr("pid", pid),
			log.AnyAttr("indices", indices),
		)
		http.Error(w, "error al buscar marco por página", http.StatusInternalServerError)
		return
	}

	// Devolver respuesta en formato JSON compatible con CPU
	response := map[string]interface{}{
		"frame": frame,
	}

	responseBytes, _ := json.Marshal(response)

	// Aplicar retardo de memoria según la cantidad de accesos a tablas (por niveles)
	time.Sleep(time.Duration(h.Config.MemoryDelay*len(indices)) * time.Millisecond)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(responseBytes)
}

func buscarMarcoPorPaginaAux(tabla *TablasProceso, indices []int) (int, bool) {
	actual := tabla.TablasDePaginas
	for i := 0; i < len(indices); i++ {
		tabla.CantidadAccesosATablas++
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

// LlenarTablaConValores Llena el ultimo nivel de la tabla para que apunte a los frames que debe ocupar
// en caso de no ocupar todos pondra en -1 los que no use
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

// RetornarPageSizeYEntries Funcion que llama la MMU para solicitar los datos de config para luego poder hacer las traducciones
func (h *Handler) RetornarPageSizeYEntries(w http.ResponseWriter, _ *http.Request) {
	response := map[string]int{
		"page_size":        h.Config.PageSize,
		"entries_per_page": h.Config.EntriesPerPage,
		"number_of_levels": h.Config.NumberOfLevels,
	}

	responseBytes, _ := json.Marshal(response)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(responseBytes)
}

// Función auxiliar para calcular potencia entera
func pow(base, exp int) int {
	if exp == 0 {
		return 1
	}
	result := 1
	for i := 0; i < exp; i++ {
		result *= base
	}
	return result
}
