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
	TLB        *TLB
	Cache      *Cache
	Log        *slog.Logger
	PageSize   int
	TLBMutex   *sync.RWMutex
	CacheMutex *sync.RWMutex
}

// TLB representa la Translation Lookaside Buffer
type TLB struct {
	Entries    map[string]*TLBEntry
	MaxEntries int
	Algoritmo  string // "FIFO" o "LRU"
}

// TLBEntry representa una entrada en la TLB
type TLBEntry struct {
	VirtualPage     string
	PhysicalPage    string
	UltimoAcceso    time.Time
	TiempoCreacion  time.Time // Para algoritmo FIFO
	ConteoDeAccesos int
}

// Cache representa la caché de páginas
type Cache struct {
	Entries    map[string]*CacheEntry
	MaxEntries int
	Algorithm  string // "CLOCK" o "CLOCK-M"
	Clock      int    // Para algoritmo CLOCK
}

// CacheEntry representa una entrada en la caché
type CacheEntry struct {
	PageID     string
	Data       string
	LastAccess time.Time
	Reference  bool // Para algoritmo CLOCK
	Modified   bool // Para algoritmo CLOCK-M
}

// NewMMU crea una nueva instancia de MMU
func NewMMU(tlbEntries, cacheEntries int, tlbAlgoritmo, cacheAlgoritmo string, logger *slog.Logger) *MMU {
	return &MMU{
		TLB: &TLB{
			Entries:    make(map[string]*TLBEntry),
			MaxEntries: tlbEntries,
			Algoritmo:  tlbAlgoritmo,
		},
		Cache: &Cache{
			Entries:    make(map[string]*CacheEntry),
			MaxEntries: cacheEntries,
			Algorithm:  cacheAlgoritmo,
			Clock:      0,
		},
		Log:        logger,
		PageSize:   64, // Tamaño de página por defecto
		TLBMutex:   &sync.RWMutex{},
		CacheMutex: &sync.RWMutex{},
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

	// Calcular número de página virtual
	nroPagina := dirLogicaInt / m.PageSize
	nroPaginaStr := strconv.Itoa(nroPagina)

	// 1. PRIMERO: Verificar caché de páginas (si está habilitada)
	if m.Cache.MaxEntries > 0 {
		pageID := m.generarPageID(pid, dirLogica)
		m.CacheMutex.RLock()
		cacheEntry, exists := m.Cache.Entries[pageID]
		m.CacheMutex.RUnlock()

		if exists {
			// Log obligatorio: Página encontrada en Caché
			// "PID: <PID> - Cache Hit - Pagina: <NUMERO_PAGINA>"
			m.Log.Info(fmt.Sprintf("PID: %d - Cache Hit - Pagina: %s", pid, nroPaginaStr))

			// Actualizar estadísticas de caché
			cacheEntry.LastAccess = time.Now()
			cacheEntry.Reference = true

			// Calcular offset y dirección física
			offset := dirLogicaInt % m.PageSize
			dirFisica := (nroPagina * m.PageSize) + offset

			return strconv.Itoa(dirFisica), nil
		} else {
			// Log obligatorio: Página faltante en Caché
			// "PID: <PID> - Cache Miss - Pagina: <NUMERO_PAGINA>"
			m.Log.Info(fmt.Sprintf("PID: %d - Cache Miss - Pagina: %s", pid, nroPaginaStr))
		}
	}

	// 2. SEGUNDO: Verificar TLB (si está habilitada)
	if m.TLB.MaxEntries > 0 {
		m.TLBMutex.RLock()
		tlbEntry, exists := m.TLB.Entries[nroPaginaStr]
		m.TLBMutex.RUnlock()

		if exists {
			// Log obligatorio: TLB Hit
			// "PID: <PID> - TLB HIT - Pagina: <NUMERO_PAGINA>"
			m.Log.Info(fmt.Sprintf("PID: %d - TLB HIT - Pagina: %s", pid, nroPaginaStr))

			// Actualizar estadísticas de TLB
			tlbEntry.UltimoAcceso = time.Now()
			tlbEntry.ConteoDeAccesos++

			// Calcular offset (desplazamiento) y dirección física
			offset := dirLogicaInt % m.PageSize
			dirFisica := (nroPagina * m.PageSize) + offset

			// Log obligatorio: Obtener Marco
			// "PID: <PID> - OBTENER MARCO - Página: <NUMERO_PAGINA> - Marco: <NUMERO_MARCO>"
			m.Log.Info(fmt.Sprintf("PID: %d - OBTENER MARCO - Página: %s - Marco: %s", pid, nroPaginaStr, tlbEntry.PhysicalPage))

			return strconv.Itoa(dirFisica), nil
		} else {
			// Log obligatorio: TLB Miss
			// "PID: <PID> - TLB MISS - Pagina: <NUMERO_PAGINA>"
			m.Log.Info(fmt.Sprintf("PID: %d - TLB MISS - Pagina: %s", pid, nroPaginaStr))
		}
	}

	// 3. TERCERO: Consultar tabla de páginas en memoria
	// TLB miss - por ahora usamos traducción directa
	// TODO: Implementar tabla de páginas real consultando a Memoria
	m.Log.Debug("TLB miss, usando traducción directa",
		log.IntAttr("pid", pid),
		log.StringAttr("nro_pagina", nroPaginaStr),
	)

	// TODO: Replace later
	// Simular obtención de marco de tabla de páginas
	marco := nroPagina // Por simplicidad, el marco es igual al número de página
	marcoStr := strconv.Itoa(marco)

	// Log obligatorio: Obtener Marco desde tabla de páginas
	// "PID: <PID> - OBTENER MARCO - Página: <NUMERO_PAGINA> - Marco: <NUMERO_MARCO>"
	m.Log.Info(fmt.Sprintf("PID: %d - OBTENER MARCO - Página: %s - Marco: %s", pid, nroPaginaStr, marcoStr))

	// Agregar entrada a TLB si está habilitada
	if m.TLB.MaxEntries > 0 {
		m.agregarATLB(nroPaginaStr, marcoStr)
	}

	// Por ahora, la dirección física es igual a la lógica
	return dirLogica, nil
}

// LeerConCache realiza una operación de lectura usando la caché si está habilitada
func (m *MMU) LeerConCache(pid int, direccion string, tamanio int, memoriaClient *memoria.Memoria) (string, error) {
	// Calcular número de página
	dirLogicaInt, err := strconv.Atoi(direccion)
	if err != nil {
		return "", err
	}
	nroPagina := dirLogicaInt / m.PageSize
	nroPaginaStr := strconv.Itoa(nroPagina)

	// Verificar si la caché está habilitada
	if m.Cache.MaxEntries == 0 {
		m.Log.Debug("Caché deshabilitada, accediendo directamente a memoria",
			log.IntAttr("pid", pid),
			log.StringAttr("direccion", direccion))

		// Acceso directo a memoria
		return memoriaClient.Read(pid, direccion, tamanio)
	}

	// Generar ID de página para la caché
	pageID := m.generarPageID(pid, direccion)

	// Buscar en caché primero
	m.CacheMutex.RLock()
	cacheEntry, exists := m.Cache.Entries[pageID]
	m.CacheMutex.RUnlock()

	if exists {
		// Log obligatorio: Página encontrada en Caché
		// "PID: <PID> - Cache Hit - Pagina: <NUMERO_PAGINA>"
		m.Log.Info(fmt.Sprintf("PID: %d - Cache Hit - Pagina: %s", pid, nroPaginaStr))

		// Actualizar estadísticas de caché
		cacheEntry.LastAccess = time.Now()
		cacheEntry.Reference = true

		// Retornar datos de la caché
		return cacheEntry.Data, nil
	}

	// Log obligatorio: Página faltante en Caché
	// "PID: <PID> - Cache Miss - Pagina: <NUMERO_PAGINA>"
	m.Log.Info(fmt.Sprintf("PID: %d - Cache Miss - Pagina: %s", pid, nroPaginaStr))

	// Cache miss - leer de memoria
	m.Log.Debug("Cache miss en lectura, accediendo a memoria",
		log.IntAttr("pid", pid),
		log.StringAttr("direccion", direccion),
		log.StringAttr("page_id", pageID))

	datos, err := memoriaClient.Read(pid, direccion, tamanio)
	if err != nil {
		return "", err
	}

	// Agregar a caché
	m.agregarACache(pageID, datos)

	// Log obligatorio: Página ingresada en Caché
	// "PID: <PID> - Cache Add - Pagina: <NUMERO_PAGINA>"
	m.Log.Info(fmt.Sprintf("PID: %d - Cache Add - Pagina: %s", pid, nroPaginaStr))

	return datos, nil
}

// EscribirConCache realiza una operación de escritura usando la caché si está habilitada
func (m *MMU) EscribirConCache(pid int, direccion string, datos string, memoriaClient *memoria.Memoria) error {
	// Calcular número de página
	dirLogicaInt, err := strconv.Atoi(direccion)
	if err != nil {
		return err
	}
	nroPagina := dirLogicaInt / m.PageSize
	nroPaginaStr := strconv.Itoa(nroPagina)

	// Verificar si la caché está habilitada
	if m.Cache.MaxEntries == 0 {
		m.Log.Debug("Caché deshabilitada, escribiendo directamente a memoria",
			log.IntAttr("pid", pid),
			log.StringAttr("direccion", direccion))

		// Acceso directo a memoria
		return memoriaClient.Write(pid, direccion, datos)
	}

	// Generar ID de página para la caché
	pageID := m.generarPageID(pid, direccion)

	// Buscar en caché primero
	m.CacheMutex.Lock()
	cacheEntry, exists := m.Cache.Entries[pageID]

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

		// Para simplificar, escribimos inmediatamente a memoria también
		// En una implementación real, esto se haría en el momento de evicción
		err = memoriaClient.Write(pid, direccion, datos)
		if err == nil {
			// Log obligatorio: Página Actualizada de Caché a Memoria
			// "PID: <PID> - Memory Update - Página: <NUMERO_PAGINA> - Frame: <FRAME_EN_MEMORIA_PRINCIPAL>"
			frame := nroPagina // Por simplicidad, el frame es igual al número de página
			m.Log.Info(fmt.Sprintf("PID: %d - Memory Update - Página: %s - Frame: %d", pid, nroPaginaStr, frame))
		}
		return err
	}

	// Log obligatorio: Página faltante en Caché
	// "PID: <PID> - Cache Miss - Pagina: <NUMERO_PAGINA>"
	m.Log.Info(fmt.Sprintf("PID: %d - Cache Miss - Pagina: %s", pid, nroPaginaStr))

	// Cache miss - agregar nueva entrada
	m.Cache.Entries[pageID] = &CacheEntry{
		PageID:     pageID,
		Data:       datos,
		LastAccess: time.Now(),
		Reference:  true,
		Modified:   true,
	}
	m.CacheMutex.Unlock()

	// Log obligatorio: Página ingresada en Caché
	// "PID: <PID> - Cache Add - Pagina: <NUMERO_PAGINA>"
	m.Log.Info(fmt.Sprintf("PID: %d - Cache Add - Pagina: %s", pid, nroPaginaStr))

	// Escribir a memoria
	err = memoriaClient.Write(pid, direccion, datos)
	if err == nil {
		// Log obligatorio: Página Actualizada de Caché a Memoria
		// "PID: <PID> - Memory Update - Página: <NUMERO_PAGINA> - Frame: <FRAME_EN_MEMORIA_PRINCIPAL>"
		frame := nroPagina // Por simplicidad, el frame es igual al número de página
		m.Log.Info(fmt.Sprintf("PID: %d - Memory Update - Página: %s - Frame: %d", pid, nroPaginaStr, frame))
	}

	// Verificar si necesitamos hacer evicción
	if len(m.Cache.Entries) > m.Cache.MaxEntries {
		m.evictCacheEntry()
	}

	return err
}

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
			log.StringAttr("key", key),
			log.StringAttr("physical_page", entry.PhysicalPage))
	}
	m.TLBMutex.Unlock()

	// Limpiar caché - escribir páginas modificadas a memoria y eliminar entradas del proceso
	m.CacheMutex.Lock()
	for key, entry := range m.Cache.Entries {
		// Limpiamos toda la caché
		if entry.Modified {
			m.Log.Debug("Escribiendo página modificada a memoria antes de limpiar",
				log.StringAttr("page_id", entry.PageID))
			// Aquí se escribiría a memoria si fuera necesario
		}
		delete(m.Cache.Entries, key)
		m.Log.Debug("Entrada caché eliminada",
			log.StringAttr("page_id", entry.PageID))
	}
	m.CacheMutex.Unlock()

	m.Log.Debug("Limpieza de memoria completada",
		log.IntAttr("pid", pid))
}

// agregarATLB agrega una nueva entrada a la TLB
func (m *MMU) agregarATLB(nroPagina, marco string) {
	m.TLBMutex.Lock()
	defer m.TLBMutex.Unlock()

	// Verificar si necesitamos hacer evicción
	if len(m.TLB.Entries) >= m.TLB.MaxEntries {
		m.evictTLBEntry()
	}

	// Agregar nueva entrada
	m.TLB.Entries[nroPagina] = &TLBEntry{
		VirtualPage:     nroPagina,
		PhysicalPage:    marco,
		UltimoAcceso:    time.Now(),
		TiempoCreacion:  time.Now(),
		ConteoDeAccesos: 1,
	}

	m.Log.Debug("Nueva entrada agregada a TLB",
		log.StringAttr("nro_pagina", nroPagina),
		log.StringAttr("marco", marco))
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

// generarPageID genera un ID único para una página en la caché
func (m *MMU) generarPageID(pid int, direccion string) string {
	dirLogicaInt, _ := strconv.Atoi(direccion)
	nroPagina := dirLogicaInt / m.PageSize
	return fmt.Sprintf("%d_%d", pid, nroPagina)
}

// agregarACache agrega una nueva entrada a la caché
func (m *MMU) agregarACache(pageID, data string) {
	m.CacheMutex.Lock()
	defer m.CacheMutex.Unlock()

	// Verificar si necesitamos hacer evicción
	if len(m.Cache.Entries) >= m.Cache.MaxEntries {
		m.evictCacheEntry()
	}

	// Agregar nueva entrada
	m.Cache.Entries[pageID] = &CacheEntry{
		PageID:     pageID,
		Data:       data,
		LastAccess: time.Now(),
		Reference:  true,
		Modified:   false,
	}

	m.Log.Debug("Nueva entrada agregada a caché",
		log.StringAttr("page_id", pageID))
}

// evictCacheEntry remueve una entrada de la caché según el algoritmo configurado
func (m *MMU) evictCacheEntry() {
	if m.Cache.Algorithm == "CLOCK" {
		m.evictCacheClock()
	} else if m.Cache.Algorithm == "CLOCK-M" {
		m.evictCacheClockM()
	}
}

// evictCacheClock implementa el algoritmo CLOCK para caché
func (m *MMU) evictCacheClock() {
	keys := make([]string, 0, len(m.Cache.Entries))
	for key := range m.Cache.Entries {
		keys = append(keys, key)
	}

	if len(keys) > 0 {
		// Buscar una página con reference bit = false
		for _, key := range keys {
			entry := m.Cache.Entries[key]
			if !entry.Reference {
				delete(m.Cache.Entries, key)
				m.Log.Debug("Entrada caché evictada (CLOCK)",
					log.StringAttr("page_id", key))
				return
			}
			entry.Reference = false // Limpiar reference bit
		}

		// Si todas tenían reference bit = true, remover la primera
		if len(keys) > 0 {
			firstKey := keys[0]
			delete(m.Cache.Entries, firstKey)
			m.Log.Debug("Entrada caché evictada (CLOCK - segunda pasada)",
				log.StringAttr("page_id", firstKey))
		}
	}
}

// evictCacheClockM implementa el algoritmo CLOCK modificado para caché
func (m *MMU) evictCacheClockM() {
	keys := make([]string, 0, len(m.Cache.Entries))
	for key := range m.Cache.Entries {
		keys = append(keys, key)
	}

	if len(keys) > 0 {
		// Priorizar páginas no modificadas y no referenciadas
		for _, key := range keys {
			entry := m.Cache.Entries[key]
			if !entry.Reference && !entry.Modified {
				delete(m.Cache.Entries, key)
				m.Log.Debug("Entrada caché evictada (CLOCK-M)",
					log.StringAttr("page_id", key))
				return
			}
		}

		// Si no hay páginas ideales, usar la primera
		if len(keys) > 0 {
			firstKey := keys[0]
			delete(m.Cache.Entries, firstKey)
			m.Log.Debug("Entrada caché evictada (CLOCK-M - fallback)",
				log.StringAttr("page_id", firstKey))
		}
	}
}
