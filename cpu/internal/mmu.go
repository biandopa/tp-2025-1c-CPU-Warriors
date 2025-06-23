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

	// 1. PRIMERO: Verificar caché de páginas
	if m.CacheHabilitada() {
		pageID := m.generarPageID(pid, dirLogica)
		m.CacheMutex.RLock()
		cacheEntry, exists := m.Cache.Entries[pageID]
		m.CacheMutex.RUnlock()

		if exists {
			m.Log.Debug("Cache hit en traducción de dirección",
				log.IntAttr("pid", pid),
				log.StringAttr("nro_pagina", nroPaginaStr),
				log.StringAttr("page_id", pageID))

			// Actualizar estadísticas de caché
			cacheEntry.LastAccess = time.Now()
			cacheEntry.Reference = true

			// Calcular offset y dirección física
			offset := dirLogicaInt % m.PageSize
			dirFisica := (nroPagina * m.PageSize) + offset

			return strconv.Itoa(dirFisica), nil
		}
	}

	// 2. SEGUNDO: Verificar TLB
	// Si la TLB está habilitada, buscar en ella
	if m.TLBHabilitada() {
		m.TLBMutex.RLock()
		tlbEntry, exists := m.TLB.Entries[nroPaginaStr]
		m.TLBMutex.RUnlock()

		if exists {
			// TLB hit
			m.Log.Debug("TLB hit",
				log.IntAttr("pid", pid),
				log.StringAttr("nro_pagina", nroPaginaStr),
				log.StringAttr("pagina_fisica", tlbEntry.PhysicalPage),
			)

			// Actualizar estadísticas de TLB
			tlbEntry.UltimoAcceso = time.Now()
			tlbEntry.ConteoDeAccesos++

			// Calcular offset (desplazamiento) y dirección física
			offset := dirLogicaInt % m.PageSize
			dirFisica := (nroPagina * m.PageSize) + offset

			return strconv.Itoa(dirFisica), nil
		}
	}

	// 3. TERCERO: Consultar tabla de páginas en memoria
	// TLB miss - por ahora usamos traducción directa
	// TODO: Implementar tabla de páginas real consultando a Memoria
	m.Log.Debug("TLB miss, usando traducción directa",
		log.IntAttr("pid", pid),
		log.StringAttr("nro_pagina", nroPaginaStr),
	)

	// Por ahora, la dirección física es igual a la lógica
	return dirLogica, nil
}

// AddTLBEntry agrega una entrada a la TLB
func (m *MMU) AddTLBEntry(virtualPage, pagFisica string) {
	m.TLBMutex.Lock()
	defer m.TLBMutex.Unlock()

	// Si la TLB está llena, aplicar algoritmo de reemplazo
	if len(m.TLB.Entries) >= m.TLB.MaxEntries {
		m.evictTLBEntry()
	}

	// Agregar nueva entrada
	m.TLB.Entries[virtualPage] = &TLBEntry{
		VirtualPage:     virtualPage,
		PhysicalPage:    pagFisica,
		UltimoAcceso:    time.Now(),
		TiempoCreacion:  time.Now(),
		ConteoDeAccesos: 1,
	}

	m.Log.Debug("Entrada agregada a TLB",
		log.StringAttr("virtual_page", virtualPage),
		log.StringAttr("pagina_fisica", pagFisica),
	)
}

// evictTLBEntry elimina una entrada de la TLB según el algoritmo configurado
func (m *MMU) evictTLBEntry() {
	switch m.TLB.Algoritmo {
	case "LRU":
		m.evictTLBLRU()
	default:
		m.evictTLBFIFO() // Por defecto FIFO
	}
}

// evictTLBFIFO elimina la entrada más antigua de la TLB (primera en llegar)
func (m *MMU) evictTLBFIFO() {
	var (
		oldestEntry string
		oldestTime  time.Time
	)

	for virtualPage, entry := range m.TLB.Entries {
		if oldestEntry == "" || entry.TiempoCreacion.Before(oldestTime) {
			oldestEntry = virtualPage
			oldestTime = entry.TiempoCreacion
		}
	}

	if oldestEntry != "" {
		delete(m.TLB.Entries, oldestEntry)
		m.Log.Debug("Entrada eliminada de TLB (FIFO)",
			log.StringAttr("virtual_page", oldestEntry),
			log.StringAttr("tiempo_creacion", oldestTime.Format("15:04:05.000")),
		)
	}
}

// evictTLBLRU elimina la entrada menos usada recientemente
func (m *MMU) evictTLBLRU() {
	var (
		lruEntry string
		lruTime  time.Time
	)

	for virtualPage, entry := range m.TLB.Entries {
		if lruEntry == "" || entry.UltimoAcceso.Before(lruTime) {
			lruEntry = virtualPage
			lruTime = entry.UltimoAcceso
		}
	}

	if lruEntry != "" {
		delete(m.TLB.Entries, lruEntry)
		m.Log.Debug("Entrada eliminada de TLB (LRU)",
			log.StringAttr("virtual_page", lruEntry),
			log.StringAttr("ultimo_acceso", lruTime.Format("15:04:05.000")),
		)
	}
}

// AddCacheEntry agrega una entrada a la caché
func (m *MMU) AddCacheEntry(pageID, data string) {
	m.CacheMutex.Lock()
	defer m.CacheMutex.Unlock()

	// Si la caché está llena, aplicar algoritmo de reemplazo
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

	m.Log.Debug("Entrada agregada a caché",
		log.StringAttr("page_id", pageID),
	)
}

// evictCacheEntry elimina una entrada de la caché según el algoritmo configurado
// Si la página fue modificada, la escribe a memoria antes de eliminarla
func (m *MMU) evictCacheEntry() {
	switch m.Cache.Algorithm {
	case "CLOCK":
		m.evictCacheClock()
	case "CLOCK-M":
		m.evictCacheClockM()
	default:
		m.evictCacheClock() // Por defecto CLOCK
	}
}

// evictCacheClock implementa el algoritmo CLOCK para la caché
func (m *MMU) evictCacheClock() {
	// Implementación simplificada del algoritmo CLOCK
	for pageID, entry := range m.Cache.Entries {
		if !entry.Reference {
			// Si la página fue modificada, escribir a memoria antes de evicción
			if entry.Modified {
				m.Log.Debug("Página modificada encontrada, escribiendo a memoria antes de evicción",
					log.StringAttr("page_id", pageID))
				// TODO: Aquí se debería escribir a memoria usando el cliente de memoria
				// Por ahora solo logueamos la acción
			}

			delete(m.Cache.Entries, pageID)
			m.Log.Debug("Entrada eliminada de caché (CLOCK)",
				log.StringAttr("page_id", pageID),
				log.StringAttr("was_modified", fmt.Sprintf("%t", entry.Modified)))
			return
		}
		entry.Reference = false
	}
}

// evictCacheClockM implementa el algoritmo CLOCK-M para la caché
func (m *MMU) evictCacheClockM() {
	// Implementación simplificada del algoritmo CLOCK-M
	for pageID, entry := range m.Cache.Entries {
		if !entry.Reference && !entry.Modified {
			delete(m.Cache.Entries, pageID)
			m.Log.Debug("Entrada eliminada de caché (CLOCK-M)",
				log.StringAttr("page_id", pageID),
				log.StringAttr("was_modified", "false"))
			return
		}
		if !entry.Reference {
			entry.Reference = false
		}
	}
}

// LimpiarTLBProceso elimina todas las entradas de la TLB asociadas a un proceso específico
// Se llama cuando un proceso es desalojado del CPU
func (m *MMU) LimpiarTLBProceso(pid int) {
	m.TLBMutex.Lock()
	defer m.TLBMutex.Unlock()

	// Contar entradas antes de limpiar para logging
	entradasAntes := len(m.TLB.Entries)

	// Eliminar todas las entradas de la TLB
	// Limpiamos toda la TLB ya que cada CPU maneja un proceso a la vez
	m.TLB.Entries = make(map[string]*TLBEntry)

	entradasEliminadas := entradasAntes - len(m.TLB.Entries)

	m.Log.Debug("TLB limpiada por desalojo de proceso",
		log.IntAttr("pid", pid),
		log.IntAttr("entradas_eliminadas", entradasEliminadas),
		log.IntAttr("entradas_restantes", len(m.TLB.Entries)),
	)
}

// LimpiarCacheProceso elimina todas las entradas de la caché asociadas a un proceso específico
// Si hay páginas modificadas, las escribe a memoria antes de eliminar
func (m *MMU) LimpiarCacheProceso(pid int) {
	m.CacheMutex.Lock()
	defer m.CacheMutex.Unlock()

	// Contar entradas antes de limpiar para logging
	entradasAntes := len(m.Cache.Entries)
	paginasModificadas := 0

	// Verificar páginas modificadas y escribirlas a memoria antes de eliminar
	for pageID, entry := range m.Cache.Entries {
		if entry.Modified {
			m.Log.Debug("Página modificada encontrada, escribiendo a memoria antes de limpiar caché",
				log.StringAttr("page_id", pageID),
				log.IntAttr("pid", pid))
			// TODO: Aquí se debería escribir a memoria usando el cliente de memoria
			// Por ahora solo contamos las páginas modificadas
			paginasModificadas++
		}
	}

	// Eliminar todas las entradas de la caché
	// Por ahora, limpiamos toda la caché, ya que cada CPU maneja un proceso a la vez
	m.Cache.Entries = make(map[string]*CacheEntry)

	entradasEliminadas := entradasAntes - len(m.Cache.Entries)

	m.Log.Debug("Caché limpiada por desalojo de proceso",
		log.IntAttr("pid", pid),
		log.IntAttr("entradas_eliminadas", entradasEliminadas),
		log.IntAttr("paginas_modificadas_escritas", paginasModificadas),
		log.IntAttr("entradas_restantes", len(m.Cache.Entries)))
}

// LimpiarMemoriaProceso elimina todas las entradas de memoria asociadas a un proceso
// Incluye TLB y caché
func (m *MMU) LimpiarMemoriaProceso(pid int) {
	m.Log.Debug("Iniciando limpieza de memoria por desalojo de proceso",
		log.IntAttr("pid", pid))

	m.LimpiarTLBProceso(pid)
	m.LimpiarCacheProceso(pid)

	m.Log.Debug("Limpieza de memoria completada",
		log.IntAttr("pid", pid))
}

// TLBHabilitada verifica si la TLB está habilitada (tiene al menos 1 entrada disponible)
func (m *MMU) TLBHabilitada() bool {
	m.TLBMutex.RLock()
	defer m.TLBMutex.RUnlock()

	// La TLB está habilitada si tiene al menos 1 entrada disponible
	return m.TLB.MaxEntries > 0
}

// CacheHabilitada verifica si la caché está habilitada (tiene al menos 1 frame disponible)
func (m *MMU) CacheHabilitada() bool {
	m.CacheMutex.RLock()
	defer m.CacheMutex.RUnlock()

	// La caché está habilitada si tiene al menos 1 frame disponible
	return m.Cache.MaxEntries > 0
}

// LeerConCache realiza una operación de lectura usando la caché si está habilitada
func (m *MMU) LeerConCache(pid int, direccion string, tamanio int, memoriaClient *memoria.Memoria) (string, error) {
	// Verificar si la caché está habilitada
	if !m.CacheHabilitada() {
		m.Log.Debug("Caché deshabilitada, accediendo directamente a memoria",
			log.IntAttr("pid", pid),
			log.StringAttr("direccion", direccion))

		// Acceso directo a memoria
		return m.leerMemoriaDirecta(pid, direccion, tamanio, memoriaClient)
	}

	// Generar ID de página para la caché
	pageID := m.generarPageID(pid, direccion)

	// Buscar en caché primero
	m.CacheMutex.RLock()
	cacheEntry, exists := m.Cache.Entries[pageID]
	m.CacheMutex.RUnlock()

	if exists {
		// Cache hit
		m.Log.Debug("Cache hit en lectura",
			log.IntAttr("pid", pid),
			log.StringAttr("direccion", direccion),
			log.StringAttr("page_id", pageID))

		// Actualizar estadísticas de caché
		cacheEntry.LastAccess = time.Now()
		cacheEntry.Reference = true

		// Retornar datos de la caché
		return cacheEntry.Data, nil
	}

	// Cache miss - leer de memoria
	m.Log.Debug("Cache miss en lectura, accediendo a memoria",
		log.IntAttr("pid", pid),
		log.StringAttr("direccion", direccion),
		log.StringAttr("page_id", pageID))

	datos, err := m.leerMemoriaDirecta(pid, direccion, tamanio, memoriaClient)
	if err != nil {
		return "", err
	}

	// Agregar a caché
	m.AddCacheEntry(pageID, datos)

	return datos, nil
}

// EscribirConCache realiza una operación de escritura usando la caché si está habilitada
func (m *MMU) EscribirConCache(pid int, direccion string, datos string, memoriaClient *memoria.Memoria) error {
	// Verificar si la caché está habilitada
	if !m.CacheHabilitada() {
		m.Log.Debug("Caché deshabilitada, escribiendo directamente en memoria",
			log.IntAttr("pid", pid),
			log.StringAttr("direccion", direccion))

		// Escritura directa en memoria
		return m.escribirMemoriaDirecta(pid, direccion, datos, memoriaClient)
	}

	// Generar ID de página para la caché
	pageID := m.generarPageID(pid, direccion)

	// Buscar en caché primero
	m.CacheMutex.RLock()
	cacheEntry, exists := m.Cache.Entries[pageID]
	m.CacheMutex.RUnlock()

	if exists {
		// Cache hit - actualizar caché
		m.Log.Debug("Cache hit en escritura, actualizando caché",
			log.IntAttr("pid", pid),
			log.StringAttr("direccion", direccion),
			log.StringAttr("page_id", pageID))

		cacheEntry.Data = datos
		cacheEntry.LastAccess = time.Now()
		cacheEntry.Reference = true
		cacheEntry.Modified = true // Marcar como modificada

		// También escribir en memoria (write-through)
		return m.escribirMemoriaDirecta(pid, direccion, datos, memoriaClient)
	}

	// Cache miss - escribir en memoria y agregar a caché
	m.Log.Debug("Cache miss en escritura, escribiendo en memoria y agregando a caché",
		log.IntAttr("pid", pid),
		log.StringAttr("direccion", direccion),
		log.StringAttr("page_id", pageID))

	err := m.escribirMemoriaDirecta(pid, direccion, datos, memoriaClient)
	if err != nil {
		return err
	}

	// Agregar a caché
	m.AddCacheEntry(pageID, datos)

	return nil
}

// generarPageID genera un ID único para una página en la caché
func (m *MMU) generarPageID(pid int, direccion string) string {
	return fmt.Sprintf("%d_%s", pid, direccion)
}

// leerMemoriaDirecta realiza lectura directa a memoria (sin caché)
func (m *MMU) leerMemoriaDirecta(pid int, direccion string, tamanio int, memoriaClient *memoria.Memoria) (string, error) {
	return memoriaClient.Read(pid, direccion, tamanio)
}

// escribirMemoriaDirecta realiza escritura directa a memoria (sin caché)
func (m *MMU) escribirMemoriaDirecta(pid int, direccion string, datos string, memoriaClient *memoria.Memoria) error {
	return memoriaClient.Write(pid, direccion, datos)
}
