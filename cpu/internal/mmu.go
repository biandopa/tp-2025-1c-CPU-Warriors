package internal

import (
	"fmt"
	"log/slog"
	"strconv"
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
}

// TLB representa la Translation Lookaside Buffer
type TLB struct {
	Entries    map[int]*TLBEntry
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
	Entries    map[int]*CacheEntry
	MaxEntries int
	Algorithm  string // "CLOCK" o "CLOCK-M"
	Clock      int    // Para algoritmo CLOCK
}

// CacheEntry representa una entrada en la caché
type CacheEntry struct {
	PID        int // PID del proceso
	PageID     int
	Data       string // Eso se pasa a memoria
	LastAccess time.Time
	Reference  bool // Para algoritmo CLOCK
	Modified   bool // Para algoritmo CLOCK-M
}

// NewMMU crea una nueva instancia de MMU
func NewMMU(tlbEntries, cacheEntries int, tlbAlgoritmo, cacheAlgoritmo string, logger *slog.Logger, memoria *memoria.Memoria) *MMU {
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
			Entries:    make(map[int]*TLBEntry),
			MaxEntries: tlbEntries,
			Algoritmo:  tlbAlgoritmo,
		},
		Cache: &Cache{
			Entries:    make(map[int]*CacheEntry),
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
	var entradasNivel []int
	if m.CantEntriesMem > 0 && m.NumberOfLevels > 0 {
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

		m.Log.Debug("Entradas por nivel calculadas",
			log.IntAttr("pid", pid),
			log.IntAttr("nro_pagina", nroPagina),
			log.IntAttr("cant_niveles", cantNiveles),
			log.IntAttr("cant_entradas_tabla", cantEntradasTabla),
			log.AnyAttr("entradas_nivel", entradasNivel),
		)
	}

	// 1. Primero: Verificar TLB (si está habilitada)
	if m.TLB.MaxEntries > 0 {
		m.TLBMutex.RLock()
		tlbEntry, exists := m.TLB.Entries[nroPagina]
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
	response, err := m.Memoria.BuscarFrame(dirLogicaInt, pid)
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
		m.agregarATLB(nroPagina, frame)
	}

	// Calcular dirección física
	dirFisica := (frame * m.PageSize) + offset

	return strconv.Itoa(dirFisica), nil
}

// LeerConCache realiza una operación de lectura usando la caché si está habilitada.
// Retorna el dato leído, la dirección física traducida y el número de página.
func (m *MMU) LeerConCache(pid int, dirLogica string, tamanio int) (string, string, error) {
	// Traducir dirección lógica a física para obtener el número de página
	dirFisica, err := m.TraducirDireccion(pid, dirLogica)
	if err != nil {
		return "", "", err
	}

	// Calcular número de página
	dirLogicaInt, _ := strconv.Atoi(dirLogica)
	nroPagina := dirLogicaInt / m.PageSize

	// Verificar si la caché está habilitada
	if m.Cache.MaxEntries == 0 {
		m.Log.Debug("Caché deshabilitada, accediendo directamente a memoria",
			log.IntAttr("pid", pid),
			log.StringAttr("direccion", dirFisica))

		// Acceso directo a memoria
		datoLeido, err := m.Memoria.Read(pid, dirFisica, tamanio)
		if err != nil {
			return "", dirFisica, err
		}
		return datoLeido, dirFisica, nil
	}

	// Buscar en caché primero
	m.CacheMutex.RLock()
	cacheEntry, exists := m.Cache.Entries[nroPagina]
	m.CacheMutex.RUnlock()

	if exists {
		// Log obligatorio: Página encontrada en Caché
		// "PID: <PID> - Cache Hit - Pagina: <NUMERO_PAGINA>"
		m.Log.Info(fmt.Sprintf("PID: %d - Cache Hit - Pagina: %d", pid, nroPagina))

		// Actualizar estadísticas de caché
		cacheEntry.LastAccess = time.Now()
		cacheEntry.Reference = true

		// Retornar datos de la caché
		return cacheEntry.Data, dirFisica, nil
	}

	// Log obligatorio: Página faltante en Caché
	// "PID: <PID> - Cache Miss - Pagina: <NUMERO_PAGINA>"
	m.Log.Info(fmt.Sprintf("PID: %d - Cache Miss - Pagina: %d", pid, nroPagina))

	datos, err := m.Memoria.Read(pid, dirFisica, tamanio)
	if err != nil {
		return "", dirFisica, err
	}

	// Agregar a caché
	m.agregarACache(pid, nroPagina, datos)

	// Log obligatorio: Página ingresada en Caché
	// "PID: <PID> - Cache Add - Pagina: <NUMERO_PAGINA>"
	m.Log.Info(fmt.Sprintf("PID: %d - Cache Add - Pagina: %d", pid, nroPagina))

	return datos, dirFisica, nil
}

// EscribirConCache realiza una operación de escritura usando la caché si está habilitada.
// Retorna la dirección física traducida y un error si ocurre.
func (m *MMU) EscribirConCache(pid int, dirLogica, datos string) (string, error) {
	// Traducir dirección lógica a física para obtener el número de página
	dirFisica, err := m.TraducirDireccion(pid, dirLogica)
	if err != nil {
		return "", err
	}

	// Calcular número de página
	dirLogicaInt, _ := strconv.Atoi(dirLogica)
	nroPagina := dirLogicaInt / m.PageSize
	nroPaginaStr := strconv.Itoa(nroPagina)

	// Verificar si la caché está habilitada
	if m.Cache.MaxEntries == 0 {
		m.Log.Debug("Caché deshabilitada, escribiendo directamente a memoria",
			log.IntAttr("pid", pid),
			log.StringAttr("direccion", dirFisica))

		// Acceso directo a memoria
		return dirFisica, m.Memoria.Write(pid, dirFisica, datos)
	}

	// Buscar en caché primero
	m.CacheMutex.Lock()
	cacheEntry, exists := m.Cache.Entries[nroPagina]

	if exists {
		// Log obligatorio: Página encontrada en Caché
		// "PID: <PID> - Cache Hit - Pagina: <NUMERO_PAGINA>"
		m.Log.Info(fmt.Sprintf("PID: %d - Cache Hit - Pagina: %s", pid, nroPaginaStr))

		// Actualizar datos en caché
		cacheEntry.Data = datos
		cacheEntry.LastAccess = time.Now()
		cacheEntry.Reference = true
		cacheEntry.Modified = true // Marcar como modificado
		m.CacheMutex.Unlock()

		return dirFisica, nil
	}

	// Log obligatorio: Página faltante en Caché
	// "PID: <PID> - Cache Miss - Pagina: <NUMERO_PAGINA>"
	m.Log.Info(fmt.Sprintf("PID: %d - Cache Miss - Pagina: %s", pid, nroPaginaStr))

	// Cache miss - agregar nueva entrada
	m.Cache.Entries[nroPagina] = &CacheEntry{
		PID:        pid,
		PageID:     nroPagina,
		Data:       datos,
		LastAccess: time.Now(),
		Reference:  true,
		Modified:   true,
	}
	m.CacheMutex.Unlock()

	// Log obligatorio: Página ingresada en Caché
	// "PID: <PID> - Cache Add - Pagina: <NUMERO_PAGINA>"
	m.Log.Info(fmt.Sprintf("PID: %d - Cache Add - Pagina: %s", pid, nroPaginaStr))

	// Verificar si necesitamos hacer evicción
	if len(m.Cache.Entries) > m.Cache.MaxEntries {
		m.evictCacheEntry()
	}

	return dirFisica, nil
}

// TODO: Agregar escritura en memoria de lo almacenado en caché (Usar lo de página completa)
// LimpiarMemoriaProceso limpia TLB y caché cuando un proceso termina o es desalojado
func (m *MMU) LimpiarMemoriaProceso(pid int) {
	m.Log.Debug("Limpiando memoria del proceso",
		log.IntAttr("pid", pid))

	// Limpiar TLB - eliminar todas las entradas del proceso
	m.TLBMutex.Lock()
	for key, entry := range m.TLB.Entries {
		// Limpiamos toda la TLB
		delete(m.TLB.Entries, key)
		m.Log.Debug("Entrada TLB eliminada",
			log.IntAttr("key", key),
			log.IntAttr("frame", entry.Frame))
	}
	m.TLBMutex.Unlock()

	// Limpiar caché - escribir páginas modificadas a memoria y eliminar entradas del proceso
	m.CacheMutex.Lock()
	for key, entry := range m.Cache.Entries {
		// Limpiamos toda la caché
		if entry.Modified {
			m.Log.Debug("Escribiendo página modificada a memoria antes de limpiar",
				log.IntAttr("page_id", entry.PageID))
			// Aquí se escribiría a memoria si fuera necesario
		}
		delete(m.Cache.Entries, key)
		m.Log.Debug("Entrada caché eliminada",
			log.IntAttr("page_id", entry.PageID))
	}
	m.CacheMutex.Unlock()

	m.Log.Debug("Limpieza de memoria completada",
		log.IntAttr("pid", pid))
}

// agregarATLB agrega una nueva entrada a la TLB
func (m *MMU) agregarATLB(nroPagina, marco int) {
	m.TLBMutex.Lock()
	defer m.TLBMutex.Unlock()

	// Verificar si necesitamos hacer evicción
	if len(m.TLB.Entries) >= m.TLB.MaxEntries {
		m.evictTLBEntry()
	}

	// Agregar nueva entrada
	m.TLB.Entries[nroPagina] = &TLBEntry{
		Page:            nroPagina,
		Frame:           marco,
		UltimoAcceso:    time.Now(),
		TiempoCreacion:  time.Now(),
		ConteoDeAccesos: 1,
	}

	m.Log.Debug("Nueva entrada agregada a TLB",
		log.IntAttr("pagina", nroPagina),
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
		oldestKey  int
		oldestTime = time.Now()
	)

	for key, entry := range m.TLB.Entries {
		if entry.TiempoCreacion.Before(oldestTime) {
			oldestTime = entry.TiempoCreacion
			oldestKey = key
		}
	}

	if oldestKey >= 0 {
		delete(m.TLB.Entries, oldestKey)
		m.Log.Debug("Entrada TLB evictada (FIFO)",
			log.IntAttr("key", oldestKey))
	}
}

// evictTLBLRU implementa el algoritmo LRU para TLB
func (m *MMU) evictTLBLRU() {
	var (
		lruKey  int
		lruTime = time.Now()
	)

	for key, entry := range m.TLB.Entries {
		if entry.UltimoAcceso.Before(lruTime) {
			lruTime = entry.UltimoAcceso
			lruKey = key
		}
	}

	if lruKey >= 0 {
		delete(m.TLB.Entries, lruKey)
		m.Log.Debug("Entrada TLB evictada (LRU)",
			log.IntAttr("key", lruKey))
	}
}

// agregarACache agrega una nueva entrada a la caché
func (m *MMU) agregarACache(pid, pageID int, data string) {
	m.CacheMutex.Lock()
	defer m.CacheMutex.Unlock()

	// Verificar si necesitamos hacer evicción
	if len(m.Cache.Entries) >= m.Cache.MaxEntries {
		m.evictCacheEntry()
	}

	// Agregar nueva entrada
	m.Cache.Entries[pageID] = &CacheEntry{
		PID:        pid,
		PageID:     pageID,
		Data:       data,
		LastAccess: time.Now(),
		Reference:  true,
		Modified:   false,
	}

	m.Log.Debug("Nueva entrada agregada a caché",
		log.IntAttr("pid", pid),
		log.IntAttr("page_id", pageID))
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
	keys := make([]int, 0, len(m.Cache.Entries))
	for key := range m.Cache.Entries {
		keys = append(keys, key)
	}

	dataAAlmacenar := map[int]map[string]interface{}{}
	if len(keys) > 0 {
		// Buscar una página con reference bit = false
		for _, key := range keys {
			entry := m.Cache.Entries[key]
			if !entry.Reference {
				// Se agrega la data a almacenar
				dataAAlmacenar[key] = map[string]interface{}{
					"pid":  entry.PID,
					"data": entry.Data,
				}
				delete(m.Cache.Entries, key)
				m.Log.Debug("Entrada caché evictada (CLOCK)",
					log.IntAttr("page_id", key))
				return
			}
			entry.Reference = false // Limpiar reference bit
		}

		// Si todas tenían reference bit = true, remover la primera
		if len(keys) > 0 {
			firstKey := keys[0]
			entry := m.Cache.Entries[firstKey]
			dataAAlmacenar[firstKey] = map[string]interface{}{
				"pid":  entry.PID,
				"data": entry.Data,
			}
			delete(m.Cache.Entries, firstKey)
			m.Log.Debug("Entrada caché evictada (CLOCK - segunda pasada)",
				log.IntAttr("page_id", firstKey))
		}
	}
	// Enviar información a memoria
	go func() {
		if err := m.Memoria.GuardarPagsEnMemoria(dataAAlmacenar); err != nil {
			m.Log.Error("Error al guardar páginas en memoria",
				log.ErrAttr(err),
			)
		}
	}()
}

// evictCacheClockM implementa el algoritmo CLOCK modificado para caché
func (m *MMU) evictCacheClockM() {
	keys := make([]int, 0, len(m.Cache.Entries))
	for key := range m.Cache.Entries {
		keys = append(keys, key)
	}

	dataAAlmacenar := map[int]map[string]interface{}{}
	if len(keys) > 0 {
		// Priorizar páginas no modificadas y no referenciadas
		for _, key := range keys {
			entry := m.Cache.Entries[key]
			if !entry.Reference && !entry.Modified {
				// Se agrega la data a almacenar
				dataAAlmacenar[key] = map[string]interface{}{
					"pid":  entry.PID,
					"data": entry.Data,
				}
				delete(m.Cache.Entries, key)
				m.Log.Debug("Entrada caché evictada (CLOCK-M)",
					log.IntAttr("page_id", key))
				return
			}
		}

		// Si no hay páginas ideales, usar la primera
		if len(keys) > 0 {
			firstKey := keys[0]
			entry := m.Cache.Entries[firstKey]
			dataAAlmacenar[firstKey] = map[string]interface{}{
				"pid":  entry.PID,
				"data": entry.Data,
			}
			delete(m.Cache.Entries, firstKey)
			m.Log.Debug("Entrada caché evictada (CLOCK-M - fallback)",
				log.IntAttr("page_id", firstKey))
		}
	}

	// Enviar información a memoria
	go func() {
		if err := m.Memoria.GuardarPagsEnMemoria(dataAAlmacenar); err != nil {
			m.Log.Error("Error al guardar páginas en memoria",
				log.ErrAttr(err),
			)
		}
	}()
}
