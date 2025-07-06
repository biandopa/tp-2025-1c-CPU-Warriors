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

	// 2. SEGUNDO: Verificar TLB (si está habilitada)
	if m.TLB.MaxEntries > 0 {
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

// LeerConCache realiza una operación de lectura usando la caché si está habilitada
func (m *MMU) LeerConCache(pid int, direccion string, tamanio int, memoriaClient *memoria.Memoria) (string, error) {
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

	datos, err := memoriaClient.Read(pid, direccion, tamanio)
	if err != nil {
		return "", err
	}

	// Agregar a caché
	m.agregarACache(pageID, datos)

	return datos, nil
}

// EscribirConCache realiza una operación de escritura usando la caché si está habilitada
func (m *MMU) EscribirConCache(pid int, direccion string, datos string, memoriaClient *memoria.Memoria) error {
	// Verificar si la caché está habilitada
	if m.Cache.MaxEntries == 0 {
		m.Log.Debug("Caché deshabilitada, escribiendo directamente en memoria",
			log.IntAttr("pid", pid),
			log.StringAttr("direccion", direccion))

		// Escritura directa en memoria
		return memoriaClient.Write(pid, direccion, datos)
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
		return memoriaClient.Write(pid, direccion, datos)
	}

	// Cache miss - escribir en memoria y agregar a caché
	m.Log.Debug("Cache miss en escritura, escribiendo en memoria y agregando a caché",
		log.IntAttr("pid", pid),
		log.StringAttr("direccion", direccion),
		log.StringAttr("page_id", pageID))

	err := memoriaClient.Write(pid, direccion, datos)
	if err != nil {
		return err
	}

	// Agregar a caché
	m.agregarACache(pageID, datos)

	return nil
}

// LimpiarMemoriaProceso elimina todas las entradas de memoria asociadas a un proceso
// Incluye TLB y caché
func (m *MMU) LimpiarMemoriaProceso(pid int) {
	m.Log.Debug("Iniciando limpieza de memoria por desalojo de proceso",
		log.IntAttr("pid", pid))

	// Limpiar TLB
	m.TLBMutex.Lock()
	entradasTLB := len(m.TLB.Entries)
	m.TLB.Entries = make(map[string]*TLBEntry)
	m.TLBMutex.Unlock()

	// Limpiar caché (escribir páginas modificadas antes)
	m.CacheMutex.Lock()
	entradasCache := len(m.Cache.Entries)
	paginasModificadas := 0

	// Contar páginas modificadas
	for _, entry := range m.Cache.Entries {
		if entry.Modified {
			paginasModificadas++
		}
	}

	m.Cache.Entries = make(map[string]*CacheEntry)
	m.CacheMutex.Unlock()

	m.Log.Debug("Limpieza de memoria completada",
		log.IntAttr("pid", pid),
		log.IntAttr("entradas_tlb_eliminadas", entradasTLB),
		log.IntAttr("entradas_cache_eliminadas", entradasCache),
		log.IntAttr("paginas_modificadas", paginasModificadas))
}

// generarPageID genera un ID único para una página en la caché
func (m *MMU) generarPageID(pid int, direccion string) string {
	return fmt.Sprintf("%d_%s", pid, direccion)
}

// agregarACache agrega una entrada a la caché con manejo de evicción
func (m *MMU) agregarACache(pageID, data string) {
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
		log.StringAttr("page_id", pageID))
}

// evictCacheEntry elimina una entrada de la caché según el algoritmo configurado
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
	for pageID, entry := range m.Cache.Entries {
		if !entry.Reference {
			// Si la página fue modificada, escribir a memoria antes de evicción
			if entry.Modified {
				m.Log.Debug("Página modificada encontrada, escribiendo a memoria antes de evicción",
					log.StringAttr("page_id", pageID))
				// TODO: Aquí se debería escribir a memoria usando el cliente de memoria
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
