package api

import (
	"bufio"
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

// ConsultarEspacioEInicializar recibe una consulta sobre el espacio libre en memoria.
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

func (h *Handler) AsignarMemoriaDeUsuario(paginasAOcupar int, pid string, esActualizacion bool) {

	var FramesLibres = h.MarcosLibres(paginasAOcupar)

	tabla := h.CrearTabla(h.Config.NumberOfLevels, h.Config.EntriesPerPage)

	h.LlenarTablaConValores(tabla, FramesLibres)

	h.Log.Debug("AsignarMemoriaDeUsuario",
		log.AnyAttr("tabla", tabla))

	var tablaProceso *TablasProceso

	if esActualizacion {
		for _, tp := range h.TablasProcesos {
			if tp.PID == pid {
				tp.TablasDePaginas = tabla
			}
		}
	} else {
		tablaProceso = &TablasProceso{
			PID:             pid,
			Tamanio:         paginasAOcupar * h.Config.MemorySize,
			TablasDePaginas: tabla,
		}
		//VER ESTO!! PUEDE QUE TENGA QUE IR AFUERA
		h.TablasProcesos = append(h.TablasProcesos, tablaProceso)
	}

	tablaMetricas, _ := h.BuscarProcesoPorPID(pid)
	tablaMetricas.CantidadSubidasMemoriaPrincipal++

	//Log obligatorio: Creación de Proceso
	//  “## PID: <PID> - Proceso Creado - Tamaño: <TAMAÑO>”
	h.Log.Info(fmt.Sprintf("“## PID: %s - Proceso Creado - Tamaño: %d", pid, paginasAOcupar*h.Config.MemorySize))

	//
	//BORRAR DE ACA PARA ABAJO ESTA AHORA PARA PROBAR LAS COSAS

	if esActualizacion == false {

		copy(h.EspacioDeUsuario[0:], []byte("hola"))

		//lectura, _ := h.BuscarMarcoPorPagina(tabla, []int{0, 0, 1})

		/*h.Log.Debug("BuscarMarcoPorPagina",
		log.AnyAttr("lectura", tabla))*/

		//h.PasarProcesoASwapAuxiliar(pid)

		//h.LeerPagina(0, 2, 1, pid)

		//h.EscribirPagina(marco int, offset int, valorAEscribir string, pid string)
		h.EscribirPagina(1, 0, "ey", pid)

		h.LeerPagina(1, 0, 5, pid)

		h.Log.Debug("FinalizarProcesoFuncionAuxiliar",
			log.AnyAttr("tablaMetricas", tablaMetricas.CantidadDeLectura))

		//h.FinalizarProcesoFuncionAuxiliar(pid)

		h.Log.Debug("FinalizarProcesoFuncionAuxiliar",
			log.AnyAttr("tablaMetricas", tablaMetricas.CantidadDeEscritura))

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

func (h *Handler) BuscarProcesoPorPID(pid string) (*TablasProceso, error) {
	for _, proceso := range h.TablasProcesos {
		if proceso.PID == pid {
			return proceso, nil
		}
	}
	return nil, fmt.Errorf("proceso con PID %s no encontrado", pid)
}

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
		log.AnyAttr("ObtenerMarcosValidos", marcosDelProceso))

	//iterar la lista de marcos un for, y por cada uno multiplicarlo por el sizepage

	archivoSwap, err := os.OpenFile("/home/utnso/Desktop/tp-2025-1c-CPU-Warriors/memoria/swapfile.bin", os.O_RDWR, 0644)
	if err != nil {
		panic(err)
	}
	//Para cerrarlo despues
	defer archivoSwap.Close()

	pidInt, _ := strconv.Atoi(pid)

	for i := 0; i < len(marcosDelProceso); i++ {
		h.ProcesoPorPosicionSwap = append(h.ProcesoPorPosicionSwap, pidInt)
	}

	for marco := range marcosDelProceso {
		err := h.escribirMarcoEnSwap(archivoSwap, marco)
		if err != nil {
			panic(err)
		}

		copy(h.EspacioDeUsuario[marco*h.Config.PageSize:((marco+1)*h.Config.PageSize-1)], make([]byte, h.Config.PageSize))
	}
	tablaMetricas, _ := h.BuscarProcesoPorPID(pid)
	tablaMetricas.CantidadBajadasSwap++
}

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
	/*Log obligatorio: Memory Dump
	“## PID: <PID> - Memory Dump solicitado”
	*/
	h.Log.Info(fmt.Sprintf("## PID: %s - Memory Dump solicitado”", pid))

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

func (h *Handler) ContienePIDEnSwap(pid int) bool {
	for _, valor := range h.ProcesoPorPosicionSwap {
		if valor == pid {
			return true
		}
	}
	return false
}

func (h *Handler) LeerPaginaCompleta(marco int, pid string) {
	h.LeerPagina(marco, 0, h.Config.PageSize, pid)
}

func (h *Handler) ActualizarPaginaCompleta(marco int, valorAEscribir string, pid string) {
	h.EscribirPagina(marco, 0, valorAEscribir, pid)
}

// VER EL TAMANIO A LEER XQ SI ES MAS GRANDE QUE LO QUE TENGO DEVOLVERIA CARACTERES REPRESENTANDO LA POSCION VACIA CHECKEAR
func (h *Handler) LeerPagina(marco int, offset int, tamanioALeer int, pid string) string {

	//if tamanioALeer mayor a cero
	lecturaMemoria := string(h.EspacioDeUsuario[((marco * h.Config.PageSize) + offset):((marco * h.Config.PageSize) + offset + tamanioALeer + 1)])

	lecturaMemoria = h.limpiarNulos(lecturaMemoria)
	/* Log obligatorio: Escritura / lectura en espacio de usuario
	“## PID: <PID> - <Lectura> - Dir. Física: <DIRECCIÓN_FÍSICA> - Tamaño: <TAMAÑO>”*/
	h.Log.Info(fmt.Sprintf("## PID: %s - %s - Dir. Física: %d - Tamaño: %d", pid, lecturaMemoria, marco+offset, tamanioALeer))

	tablaMetricas, _ := h.BuscarProcesoPorPID(pid)
	tablaMetricas.CantidadDeLectura++

	h.Log.Debug("LeerPagina",
		log.AnyAttr("lecturaMemoria", lecturaMemoria))
	return lecturaMemoria
}

func (h *Handler) limpiarNulos(cadena string) string {
	return strings.ReplaceAll(cadena, "\x00", "")
}

func (h *Handler) EscribirPagina(marco int, offset int, valorAEscribir string, pid string) {

	copy(h.EspacioDeUsuario[((marco*h.Config.PageSize)+offset):((marco*h.Config.PageSize)+offset)+len(valorAEscribir)], []byte(valorAEscribir))

	h.Log.Debug("EscribirPagina",
		log.AnyAttr("lecturaMemoria", h.EspacioDeUsuario))

	tablaMetricas, _ := h.BuscarProcesoPorPID(pid)
	tablaMetricas.CantidadDeEscritura++
	/* Log obligatorio: Escritura / lectura en espacio de usuario
	“## PID: <PID> - <Escritura> - Dir. Física: <DIRECCIÓN_FÍSICA> - Tamaño: <TAMAÑO>”*/
	h.Log.Info(fmt.Sprintf("## PID: %s - %s - Dir. Física: %d - Tamaño: %d", pid, valorAEscribir, marco+offset, len(valorAEscribir)))

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

func (h *Handler) AccesoATabla(w http.ResponseWriter, r *http.Request) {

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
	tabla, _ := h.BuscarProcesoPorPID(pid)
	//TO DO:
	//CORREGIR VER COMO LO PASA LA MMU
	h.BuscarMarcoPorPagina(tabla, []int{0, 0, 1})

}

func (h *Handler) BuscarMarcoPorPagina(tabla *TablasProceso, indices []int) (int, bool) {
	actual := tabla.TablasDePaginas
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
		tabla.CantidadAccesosATablas++
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
