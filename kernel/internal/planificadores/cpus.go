package planificadores

import (
	"github.com/sisoputnfrba/tp-golang/kernel/pkg/cpu"
	"github.com/sisoputnfrba/tp-golang/utils/log"
)

func (p *Service) AddCpuConectada(cpuId *CpuIdentificacion) {
	newCPU := cpu.NewCpu(cpuId.IP, cpuId.Puerto, cpuId.ID, p.Log)
	// Agregar la CPU a la lista de CPU conectadas
	p.CPUsConectadas = append(p.CPUsConectadas, newCPU)

	// Agregar un token al semáforo para indicar que hay una CPU más disponible
	p.CPUSemaphore <- struct{}{}

	p.Log.Debug("CPU conectada y agregada al pool",
		log.StringAttr("cpu_id", cpuId.ID),
		log.StringAttr("cpu_ip", cpuId.IP),
		log.IntAttr("cpu_puerto", cpuId.Puerto),
		log.IntAttr("cpus_disponibles", p.CantidadDeCpusDisponibles()))
}

// BuscarCPUDisponible adquiere una CPU del pool de CPUs disponibles.
// Bloquea hasta que haya una CPU disponible
func (p *Service) BuscarCPUDisponible() *cpu.Cpu {
	// Esperar hasta que haya una CPU disponible (acquire semáforo)
	<-p.CPUSemaphore

	// Buscar y reservar una CPU libre
	p.mutexCPUsConectadas.Lock()
	defer p.mutexCPUsConectadas.Unlock()

	for i := range p.CPUsConectadas {
		if p.CPUsConectadas[i].Estado {
			p.CPUsConectadas[i].Estado = false // Marcar como ocupada
			p.Log.Debug("CPU adquirida",
				log.StringAttr("cpu_id", p.CPUsConectadas[i].ID))
			return p.CPUsConectadas[i]
		}
	}

	// Esto no debería suceder si el semáforo funciona correctamente
	p.Log.Error("Error: semáforo permitió adquirir CPU pero no hay CPUs libres")
	return nil
}

// LiberarCPU libera una CPU de vuelta al pool de CPUs disponibles
func (p *Service) LiberarCPU(cpuToRelease *cpu.Cpu) {
	p.mutexCPUsConectadas.Lock()
	defer p.mutexCPUsConectadas.Unlock()
	cpuToRelease.Estado = true // Marcar como libre

	// Liberar el semáforo (release)
	p.CPUSemaphore <- struct{}{}

	p.Log.Debug("CPU liberada",
		log.StringAttr("cpu_id", cpuToRelease.ID))
}

// IntentarBuscarCPUDisponible intenta adquirir una CPU sin bloquear
// Retorna nil si no hay CPUs disponibles
func (p *Service) IntentarBuscarCPUDisponible() *cpu.Cpu {
	select {
	case <-p.CPUSemaphore:
		// Hay CPU disponible, buscarla
		p.mutexCPUsConectadas.Lock()
		defer p.mutexCPUsConectadas.Unlock()

		for i := range p.CPUsConectadas {
			if p.CPUsConectadas[i].Estado {
				p.CPUsConectadas[i].Estado = false
				p.Log.Debug("CPU adquirida (no bloqueante)",
					log.StringAttr("cpu_id", p.CPUsConectadas[i].ID))
				return p.CPUsConectadas[i]
			}
		}

		// Esto no debería suceder, devolver el token al semáforo
		p.CPUSemaphore <- struct{}{}
		return nil
	default:
		// No hay CPUs disponibles
		return nil
	}
}

// CantidadDeCpusDisponibles retorna el número de CPUs disponibles
func (p *Service) CantidadDeCpusDisponibles() int {
	return len(p.CPUSemaphore)
}
