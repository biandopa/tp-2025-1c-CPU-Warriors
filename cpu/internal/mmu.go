package internal

import (
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/sisoputnfrba/tp-golang/cpu/pkg/memoria"
	"github.com/sisoputnfrba/tp-golang/utils/log"
)

// MMU representa la unidad de gestión de memoria
type MMU struct {
	TLB            *TLB
	Cache          *Cache
	Log            *slog.Logger
	PageSize       int
	CantEntriesMem int
	NumberOfLevels int
	TLBMutex       *sync.RWMutex
	CacheMutex     *sync.RWMutex
	Memoria        *memoria.Memoria
	Retardo        time.Duration // Retardo para operaciones de caché
}

// TLB representa la Translation Lookaside Buffer
type TLB struct {
	Entries    map[string]*TLBEntry
	MaxEntries int
	Algoritmo  string // "FIFO" o "LRU"
}

// TLBEntry representa una entrada en la TLB
type TLBEntry struct {
	Page            int
	Frame           int
	UltimoAcceso    time.Time
	TiempoCreacion  time.Time // Para algoritmo FIFO
	ConteoDeAccesos int
}

// Cache representa la caché de páginas
type Cache struct {
	Entries    []*CacheEntry
	MaxEntries int
	Algorithm  string // "CLOCK" o "CLOCK-M"
	Clock      int    // Para algoritmo CLOCK
}

// CacheEntry representa una entrada en la caché
type CacheEntry struct {
	PID        int // PID del proceso
	PageID     string
	Data       string // Eso se pasa a memoria
	LastAccess time.Time
	Reference  bool // Para algoritmo CLOCK
	Modified   bool // Para algoritmo CLOCK-M
}

// NewMMU crea una nueva instancia de MMU
func NewMMU(tlbEntries, cacheEntries int, tlbAlgoritmo, cacheAlgoritmo string, logger *slog.Logger,
	memoria *memoria.Memoria, retardoCache time.Duration) *MMU {
	// Consultar a memoria la cantidad de entradas y el tamaño de página
	info, err := memoria.ConsultarPageSize()
	if err != nil {
		logger.Error("Error al consultar información de memoria",
			log.ErrAttr(err),
		)
		panic(err)
	}

	return &MMU{
		TLB: &TLB{
			Entries:    make(map[string]*TLBEntry),
			MaxEntries: tlbEntries,
			Algoritmo:  tlbAlgoritmo,
		},
		Cache: &Cache{
			Entries:    make([]*CacheEntry, 0),
			MaxEntries: cacheEntries,
			Algorithm:  cacheAlgoritmo,
			Clock:      0,
		},
		Log:            logger,
		TLBMutex:       &sync.RWMutex{},
		CacheMutex:     &sync.RWMutex{},
		PageSize:       info.PageSize,
		CantEntriesMem: info.Entries,
		NumberOfLevels: info.NumberOfLevels,
		Memoria:        memoria,
		Retardo:        retardoCache,
	}
}

// TraducirDireccion traduce una dirección lógica a física
// Sigue el orden: Caché → TLB → Tabla de páginas
func (m *MMU) TraducirDireccion(pid int, dirLogica string) (string, error) {
	// Convertir dirección lógica a número
	dirLogicaInt, err := strconv.Atoi(dirLogica)
	if err != nil {
		return "", err
	}

	// Calcular número de página
	// nro_página = floor(dirección_lógica / tamaño_página)
	nroPagina := dirLogicaInt / m.PageSize

	// Calcular desplazamiento
	// desplazamiento = dirección_lógica % tamaño_página
	offset := dirLogicaInt % m.PageSize

	// Calcular entradas por nivel según paginación multinivel
	// entrada_nivel_X = floor(nro_página / cant_entradas_tabla ^ (N - X)) % cant_entradas_tabla
	entriesKey := m.calcularEntradasPorNivel(nroPagina)

	// 1. Primero: Verificar TLB (si está habilitada)
	if m.TLB.MaxEntries > 0 {
		m.TLBMutex.RLock()
		tlbEntry, exists := m.TLB.Entries[entriesKey]
		m.TLBMutex.RUnlock()

		if exists {
			// Log obligatorio: TLB Hit
			// "PID: <PID> - TLB HIT - Pagina: <NUMERO_PAGINA>"
			m.Log.Info(fmt.Sprintf("PID: %d - TLB HIT - Pagina: %d", pid, nroPagina))

			// Actualizar estadísticas de TLB
			tlbEntry.UltimoAcceso = time.Now()
			tlbEntry.ConteoDeAccesos++

			// Calcular dirección física
			dirFisica := (tlbEntry.Frame * m.PageSize) + offset

			// Log obligatorio: Obtener Marco
			// "PID: <PID> - OBTENER MARCO - Página: <NUMERO_PAGINA> - Marco: <NUMERO_MARCO>"
			m.Log.Info(fmt.Sprintf("PID: %d - OBTENER MARCO - Página: %d - Marco: %d", pid, nroPagina, tlbEntry.Frame))

			return strconv.Itoa(dirFisica), nil
		} else {
			// Log obligatorio: TLB Miss
			// "PID: <PID> - TLB MISS - Pagina: <NUMERO_PAGINA>"
			m.Log.Info(fmt.Sprintf("PID: %d - TLB MISS - Pagina: %d", pid, nroPagina))
		}
	}

	// 2. Segundo: Consultar tabla de páginas en memoria
	m.Log.Debug("TLB miss, usando traducción directa",
		log.IntAttr("pid", pid),
		log.IntAttr("pagina", nroPagina),
	)

	// Obtención de marco de tabla de páginas
	response, err := m.Memoria.BuscarFrame(pid, entriesKey)
	if err != nil {
		m.Log.Error("Error al buscar marco en tabla de páginas",
			log.ErrAttr(err),
			log.IntAttr("pid", pid),
		)
		return "", err
	}
	frame := response.Frame

	// Log obligatorio: Obtener Marco desde tabla de páginas
	// "PID: <PID> - OBTENER MARCO - Página: <NUMERO_PAGINA> - Marco: <NUMERO_MARCO>"
	m.Log.Info(fmt.Sprintf("PID: %d - OBTENER MARCO - Página: %d - Marco: %d", pid, nroPagina, frame))

	// Agregar entrada a TLB si está habilitada
	if m.TLB.MaxEntries > 0 {
		m.agregarATLB(entriesKey, nroPagina, frame)
	}

	// Calcular dirección física
	dirFisica := (frame * m.PageSize) + offset

	return strconv.Itoa(dirFisica), nil
}

// LeerConCache realiza una operación de lectura usando la caché si está habilitada.
// Retorna el dato leído, la dirección física traducida y el número de página.
func (m *MMU) LeerConCache(pid int, dirLogica string, tamanio int) (string, string, error) {
	time.Sleep(m.Retardo) // Simular retardo de caché

	// Traducir dirección lógica a física para obtener el número de página
	dirFisica, err := m.TraducirDireccion(pid, dirLogica)
	if err != nil {
		return "", "", err
	}

	// Calcular número de página
	dirLogicaInt, _ := strconv.Atoi(dirLogica)
	nroPagina := dirLogicaInt / m.PageSize

	entriesKey := m.calcularEntradasPorNivel(nroPagina)

	// Verificar si la caché está habilitada
	if m.Cache.MaxEntries == 0 {
		m.Log.Debug("Caché deshabilitada, accediendo directamente a memoria",
			log.IntAttr("pid", pid),
			log.StringAttr("direccion", dirFisica))

		// Acceso directo a memoria
		datoLeido, err := m.Memoria.Read(pid, dirFisica, tamanio, memoria.PageConfig{
			PageSize:       m.PageSize,
			Entries:        m.CantEntriesMem,
			NumberOfLevels: m.NumberOfLevels,
		})
		if err != nil {
			return "", dirFisica, err
		}
		return datoLeido, dirFisica, nil
	}

	// Buscar en caché primero
	m.CacheMutex.RLock()
	for _, entry := range m.Cache.Entries {
		if entry.PageID == entriesKey && entry.PID == pid {
			m.CacheMutex.RUnlock()
			// Log obligatorio: Página encontrada en Caché
			// "PID: <PID> - Cache Hit - Pagina: <NUMERO_PAGINA>"
			m.Log.Info(fmt.Sprintf("PID: %d - Cache Hit - Pagina: %d", pid, nroPagina))

			m.CacheMutex.Lock()
			// Actualizar estadísticas de caché
			entry.LastAccess = time.Now()
			entry.Reference = true
			m.CacheMutex.Unlock()

			valorALeer := entry.Data

			// Leer la cantidad de bytes solicitada
			if len(valorALeer) > tamanio {
				valorALeer = valorALeer[:tamanio]
			}

			// Retornar datos de la caché
			return valorALeer, dirFisica, nil
		}
	}
	m.CacheMutex.RUnlock()

	// Log obligatorio: Página faltante en Caché
	// "PID: <PID> - Cache Miss - Pagina: <NUMERO_PAGINA>"
	m.Log.Info(fmt.Sprintf("PID: %d - Cache Miss - Pagina: %d", pid, nroPagina))

	datos, err := m.Memoria.Read(pid, dirFisica, tamanio, memoria.PageConfig{
		PageSize:       m.PageSize,
		Entries:        m.CantEntriesMem,
		NumberOfLevels: m.NumberOfLevels,
	})
	if err != nil {
		return "", dirFisica, err
	}

	// Agregar a caché
	m.agregarACache(pid, entriesKey, datos, false)

	// Log obligatorio: Página ingresada en Caché
	// "PID: <PID> - Cache Add - Pagina: <NUMERO_PAGINA>"
	m.Log.Info(fmt.Sprintf("PID: %d - Cache Add - Pagina: %d", pid, nroPagina))

	return datos, dirFisica, nil
}

// EscribirConCache realiza una operación de escritura usando la caché si está habilitada.
// Retorna la dirección física traducida y un error si ocurre.
func (m *MMU) EscribirConCache(pid int, dirLogica, datos string) (string, error) {
	time.Sleep(m.Retardo) // Simular retardo de caché

	// Traducir dirección lógica a física para obtener el número de página
	dirFisica, err := m.TraducirDireccion(pid, dirLogica)
	if err != nil {
		return "", err
	}

	// Calcular número de página
	dirLogicaInt, _ := strconv.Atoi(dirLogica)
	nroPagina := dirLogicaInt / m.PageSize
	nroPaginaStr := strconv.Itoa(nroPagina)

	entriesKey := m.calcularEntradasPorNivel(nroPagina)

	// Verificar si la caché está habilitada
	if m.Cache.MaxEntries == 0 {
		m.Log.Debug("Caché deshabilitada, escribiendo directamente a memoria",
			log.IntAttr("pid", pid),
			log.StringAttr("direccion", dirFisica))

		// Acceso directo a memoria
		return dirFisica, m.Memoria.Write(pid, dirFisica, datos, memoria.PageConfig{
			PageSize:       m.PageSize,
			Entries:        m.CantEntriesMem,
			NumberOfLevels: m.NumberOfLevels,
		})
	}

	// Buscar en caché primero
	exists := false
	index := -1
	m.CacheMutex.RLock()
	for i, entry := range m.Cache.Entries {
		if entry.PageID == entriesKey && entry.PID == pid {
			exists = true
			index = i
			break
		}
	}
	m.CacheMutex.RUnlock()

	if exists {
		// Log obligatorio: Página encontrada en Caché
		// "PID: <PID> - Cache Hit - Pagina: <NUMERO_PAGINA>"
		m.Log.Info(fmt.Sprintf("PID: %d - Cache Hit - Pagina: %s", pid, nroPaginaStr))

		m.CacheMutex.Lock()
		// Actualizar datos en caché
		m.Cache.Entries[index].Data = datos
		m.Cache.Entries[index].LastAccess = time.Now()
		m.Cache.Entries[index].Reference = true
		m.Cache.Entries[index].Modified = true // Marcar como modificado
		m.CacheMutex.Unlock()

		return dirFisica, nil
	}

	// Log obligatorio: Página faltante en Caché
	// "PID: <PID> - Cache Miss - Pagina: <NUMERO_PAGINA>"
	m.Log.Info(fmt.Sprintf("PID: %d - Cache Miss - Pagina: %s", pid, nroPaginaStr))

	m.agregarACache(pid, entriesKey, datos, true)

	// Log obligatorio: Página ingresada en Caché
	// "PID: <PID> - Cache Add - Pagina: <NUMERO_PAGINA>"
	m.Log.Info(fmt.Sprintf("PID: %d - Cache Add - Pagina: %s", pid, nroPaginaStr))

	return dirFisica, nil
}

// LimpiarMemoriaProceso limpia TLB y caché cuando un proceso termina o es desalojado
func (m *MMU) LimpiarMemoriaProceso(pid int) {
	m.Log.Debug("Limpiando memoria del proceso",
		log.IntAttr("pid", pid))

	dataToSave := map[string]map[string]interface{}{}
	// Limpiar TLB - eliminar todas las entradas del proceso
	m.TLBMutex.Lock()
	for key, entry := range m.TLB.Entries {
		// Limpiamos toda la TLB
		delete(m.TLB.Entries, key)
		m.Log.Debug("Entrada TLB eliminada",
			log.StringAttr("key", key),
			log.IntAttr("frame", entry.Frame))
	}
	m.TLBMutex.Unlock()

	// Limpiar caché - escribir páginas modificadas a memoria y eliminar entradas del proceso
	m.CacheMutex.Lock()
	for i := len(m.Cache.Entries) - 1; i >= 0; i-- {
		entry := m.Cache.Entries[i]
		if entry.PID != pid {
			continue // Solo limpiar entradas del proceso especificado
		}
		if entry.Modified {
			m.Log.Debug("Escribiendo página modificada a memoria antes de limpiar",
				log.StringAttr("page_id", entry.PageID))
			dataToSave[entry.PageID] = map[string]interface{}{
				"pid":  strconv.Itoa(entry.PID),
				"data": entry.Data,
			}
		}
		// Eliminar entrada de caché
		m.Cache.Entries = append(m.Cache.Entries[:i], m.Cache.Entries[i+1:]...)
		m.Log.Debug("Entrada caché eliminada",
			log.StringAttr("page_id", entry.PageID))
	}
	m.CacheMutex.Unlock()

	// Enviar información a memoria

	if err := m.Memoria.GuardarPagsEnMemoria(dataToSave); err != nil {
		m.Log.Error("Error al guardar páginas en memoria",
			log.ErrAttr(err),
		)
	}

	m.Log.Debug("Limpieza de memoria completada",
		log.IntAttr("pid", pid))
}

// agregarATLB agrega una nueva entrada a la TLB
func (m *MMU) agregarATLB(entriesKey string, page, marco int) {
	m.TLBMutex.Lock()
	defer m.TLBMutex.Unlock()

	// Verificar si necesitamos hacer evicción
	if len(m.TLB.Entries) >= m.TLB.MaxEntries {
		m.evictTLBEntry()
	}

	// Agregar nueva entrada
	m.TLB.Entries[entriesKey] = &TLBEntry{
		Page:            page,
		Frame:           marco,
		UltimoAcceso:    time.Now(),
		TiempoCreacion:  time.Now(),
		ConteoDeAccesos: 1,
	}

	m.Log.Debug("Nueva entrada agregada a TLB",
		log.IntAttr("pagina", page),
		log.IntAttr("marco", marco))
}

// evictTLBEntry remueve una entrada de la TLB según el algoritmo configurado
func (m *MMU) evictTLBEntry() {
	switch m.TLB.Algoritmo {
	case "FIFO":
		m.evictTLBFIFO()
	case "LRU":
		m.evictTLBLRU()
	}
}

// evictTLBFIFO implementa el algoritmo FIFO para TLB
func (m *MMU) evictTLBFIFO() {
	var (
		oldestKey  string
		oldestTime = time.Now()
	)

	for key, entry := range m.TLB.Entries {
		if entry.TiempoCreacion.Before(oldestTime) {
			oldestTime = entry.TiempoCreacion
			oldestKey = key
		}
	}

	if oldestKey != "" {
		delete(m.TLB.Entries, oldestKey)
		m.Log.Debug("Entrada TLB evictada (FIFO)",
			log.StringAttr("key", oldestKey))
	}
}

// evictTLBLRU implementa el algoritmo LRU para TLB
func (m *MMU) evictTLBLRU() {
	var (
		lruKey  string
		lruTime = time.Now()
	)

	for key, entry := range m.TLB.Entries {
		if entry.UltimoAcceso.Before(lruTime) {
			lruTime = entry.UltimoAcceso
			lruKey = key
		}
	}

	if lruKey != "" {
		delete(m.TLB.Entries, lruKey)
		m.Log.Debug("Entrada TLB evictada (LRU)",
			log.StringAttr("key", lruKey))
	}
}

// agregarACache agrega una nueva entrada a la caché
func (m *MMU) agregarACache(pid int, entriesXPage, data string, modificado bool) {
	// Verificar si necesitamos hacer evicción
	if len(m.Cache.Entries) >= m.Cache.MaxEntries {
		m.evictCacheEntry()
	}

	m.CacheMutex.Lock()
	// Agregar nueva entrada
	m.Cache.Entries = append(m.Cache.Entries, &CacheEntry{
		PID:        pid,
		PageID:     entriesXPage,
		Data:       data,
		LastAccess: time.Now(),
		Reference:  true,
		Modified:   modificado,
	})
	m.CacheMutex.Unlock()

	m.Log.Debug("Nueva entrada agregada a caché",
		log.IntAttr("pid", pid),
		log.StringAttr("page_id", entriesXPage))
}

// evictCacheEntry remueve una entrada de la caché según el algoritmo configurado
func (m *MMU) evictCacheEntry() {
	switch m.Cache.Algorithm {
	case "CLOCK":
		m.evictCacheClock()
	case "CLOCK-M":
		m.evictCacheClockM()
	}
}

// evictCacheClock implementa el algoritmo CLOCK para caché
func (m *MMU) evictCacheClock() {
	m.CacheMutex.Lock()
	defer m.CacheMutex.Unlock()

	dataAAlmacenar := map[string]map[string]interface{}{}
	newArrayCache := m.reordenarCacheEntries()

	// La primera iteración comienza desde el puntero del CLOCK y va buscando entradas con el bit de uso en 0.
	for i, entry := range newArrayCache {
		if !entry.Reference {
			// Se agrega la data a almacenar
			dataAAlmacenar[entry.PageID] = map[string]interface{}{
				"pid":  strconv.Itoa(entry.PID),
				"data": entry.Data,
			}

			// Eliminar la entrada de la caché
			m.Cache.Entries = append(m.Cache.Entries[:i], m.Cache.Entries[i+1:]...)
			m.Log.Debug("Entrada caché evictada (CLOCK)",
				log.StringAttr("page_id", entry.PageID))

			// Enviar información a memoria y salir
			if err := m.Memoria.GuardarPagsEnMemoria(dataAAlmacenar); err != nil {
				m.Log.Error("Error al guardar páginas en memoria",
					log.ErrAttr(err),
				)
			}

			// Actualizar el puntero del CLOCK
			m.Cache.Clock = i

			return
		}
		entry.Reference = false // Limpiar reference bit
	}

	// La segunda iteración comienza desde el principio y busca entradas con el bit de uso ya setteado en 0 anteriormente.
	for i, entry := range newArrayCache {
		if !entry.Reference {
			// Se agrega la data a almacenar
			dataAAlmacenar[entry.PageID] = map[string]interface{}{
				"pid":  strconv.Itoa(entry.PID),
				"data": entry.Data,
			}

			// Eliminar la entrada de la caché
			m.Cache.Entries = append(m.Cache.Entries[:i], m.Cache.Entries[i+1:]...)
			m.Log.Debug("Entrada caché evictada (CLOCK)",
				log.StringAttr("page_id", entry.PageID))

			// Enviar información a memoria y salir
			if err := m.Memoria.GuardarPagsEnMemoria(dataAAlmacenar); err != nil {
				m.Log.Error("Error al guardar páginas en memoria",
					log.ErrAttr(err),
				)
			}

			// Actualizar el puntero del CLOCK
			m.Cache.Clock = i
			return
		}
	}
}

// evictCacheClockM implementa el algoritmo CLOCK modificado para caché
func (m *MMU) evictCacheClockM() {
	m.CacheMutex.Lock()
	defer m.CacheMutex.Unlock()

	dataAAlmacenar := map[string]map[string]interface{}{}
	newArrayCache := m.reordenarCacheEntries()

	// La primera iteración comienza desde el puntero del CLOCK y va buscando entradas con el bit
	// de uso y el bit de modificado en 0.
	for i, entry := range newArrayCache {
		if !entry.Reference && !entry.Modified {
			// Se agrega la data a almacenar
			dataAAlmacenar[entry.PageID] = map[string]interface{}{
				"pid":  strconv.Itoa(entry.PID),
				"data": entry.Data,
			}

			// Eliminar la entrada de la caché
			m.Cache.Entries = append(m.Cache.Entries[:i], m.Cache.Entries[i+1:]...)
			m.Log.Debug("Entrada caché evictada (CLOCK)",
				log.StringAttr("page_id", entry.PageID))

			// Enviar información a memoria y salir
			if err := m.Memoria.GuardarPagsEnMemoria(dataAAlmacenar); err != nil {
				m.Log.Error("Error al guardar páginas en memoria",
					log.ErrAttr(err),
				)
			}

			// Actualizar el puntero del CLOCK
			m.Cache.Clock = i
			return
		}
	}

	// La segunda iteración comienza desde el principio y busca entradas con el bit de uso ya setteado en 0 anteriormente.
	for i, entry := range newArrayCache {
		if !entry.Reference {
			// Se agrega la data a almacenar
			dataAAlmacenar[entry.PageID] = map[string]interface{}{
				"pid":  strconv.Itoa(entry.PID),
				"data": entry.Data,
			}

			// Eliminar la entrada de la caché
			m.Cache.Entries = append(m.Cache.Entries[:i], m.Cache.Entries[i+1:]...)
			m.Log.Debug("Entrada caché evictada (CLOCK)",
				log.StringAttr("page_id", entry.PageID))

			// Enviar información a memoria y salir
			if err := m.Memoria.GuardarPagsEnMemoria(dataAAlmacenar); err != nil {
				m.Log.Error("Error al guardar páginas en memoria",
					log.ErrAttr(err),
				)
			}

			// Actualizar el puntero del CLOCK
			m.Cache.Clock = i
			return
		}
		entry.Reference = false // Limpiar reference bit
	}

	// La tercera iteración busca entradas con el bit de uso en 0 (setteado anteriormente)
	for i, entry := range newArrayCache {
		if !entry.Reference {
			// Se agrega la data a almacenar
			dataAAlmacenar[entry.PageID] = map[string]interface{}{
				"pid":  strconv.Itoa(entry.PID),
				"data": entry.Data,
			}

			// Eliminar la entrada de la caché
			m.Cache.Entries = append(m.Cache.Entries[:i], m.Cache.Entries[i+1:]...)
			m.Log.Debug("Entrada caché evictada (CLOCK)",
				log.StringAttr("page_id", entry.PageID))

			// Enviar información a memoria y salir
			if err := m.Memoria.GuardarPagsEnMemoria(dataAAlmacenar); err != nil {
				m.Log.Error("Error al guardar páginas en memoria",
					log.ErrAttr(err),
				)
			}

			// Actualizar el puntero del CLOCK
			m.Cache.Clock = i
			return
		}
	}
}

func (m *MMU) calcularEntradasPorNivel(nroPagina int) string {
	entriesKey := ""
	if m.CantEntriesMem > 0 && m.NumberOfLevels > 0 {
		var entradasNivel []int
		cantNiveles := m.NumberOfLevels
		cantEntradasTabla := m.CantEntriesMem

		for x := 1; x <= cantNiveles; x++ {
			// Calcular entrada para el nivel X
			potencia := 1
			for i := 0; i < (cantNiveles - x); i++ {
				potencia *= cantEntradasTabla
			}
			entradaNivel := (nroPagina / potencia) % cantEntradasTabla
			entradasNivel = append(entradasNivel, entradaNivel)
		}

		// Split array of int into a string with commas
		// Convertir []int a []string
		strNumeros := make([]string, len(entradasNivel))
		for i, num := range entradasNivel {
			strNumeros[i] = strconv.Itoa(num)
		}

		// Unir con "-"
		entriesKey = strings.Join(strNumeros, "-")

		m.Log.Debug("Entradas por nivel calculadas",
			log.IntAttr("nro_pagina", nroPagina),
			log.IntAttr("cant_niveles", cantNiveles),
			log.IntAttr("cant_entradas_tabla", cantEntradasTabla),
			log.AnyAttr("entradas_nivel", entradasNivel),
		)
	} else {
		entriesKey = strconv.Itoa(nroPagina) // Si no hay paginación multinivel, usar el número de página directamente
	}

	return entriesKey
}

func (m *MMU) reordenarCacheEntries() []*CacheEntry {
	n := len(m.Cache.Entries)
	if m.Cache.Clock < 0 || m.Cache.Clock >= n {
		return m.Cache.Entries // Retorna el array original si el índice es inválido
	}

	// Construir un nuevo array con el orden deseado
	nuevoArray := append(m.Cache.Entries[m.Cache.Clock:], m.Cache.Entries[:m.Cache.Clock]...)
	return nuevoArray
}
