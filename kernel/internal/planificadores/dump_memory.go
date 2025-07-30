package planificadores

import (
	"fmt"
	"time"

	"github.com/sisoputnfrba/tp-golang/kernel/internal"
	"github.com/sisoputnfrba/tp-golang/utils/log"
)

// RealizarDumpMemory maneja la syscall DUMP_MEMORY
// Bloquea el proceso, solicita el dump a memoria, y luego lo desbloquea o envía a EXIT
func (p *Service) RealizarDumpMemory(pid int) {
	// Mover el proceso de EXEC a BLOCKED
	if err := p.moverProcesoExecABlocked(pid); err != nil {
		p.Log.Error("Error al mover proceso de EXEC a BLOCKED",
			log.IntAttr("pid", pid),
			log.ErrAttr(err),
		)
		return
	}

	// Realizar el dump de memoria de forma asíncrona
	go func() {
		err := p.Memoria.DumpProceso(pid)
		if err != nil {
			// Si hay error, enviar el proceso a EXIT
			p.Log.Error("Error en DUMP_MEMORY - enviando proceso a EXIT",
				log.IntAttr("pid", pid),
				log.ErrAttr(err),
			)

			/*if err = p.moverProcesoBlockedAExit(pid); err != nil {
				p.Log.Error("Error al mover proceso de BLOCKED a EXIT",
					log.IntAttr("pid", pid),
					log.ErrAttr(err),
				)
			} else {*/
			go p.FinalizarProcesoEnCualquierCola(pid)
			//}

		} else {
			// Si es exitoso, mover el proceso de BLOCKED a READY
			p.Log.Debug("DUMP_MEMORY exitoso - desbloqueando proceso",
				log.IntAttr("pid", pid),
			)
			if err = p.moverProcesoBlockedAReady(pid); err != nil {
				p.Log.Error("Error al mover proceso de BLOCKED a READY",
					log.IntAttr("pid", pid),
					log.ErrAttr(err),
				)
			}
		}
	}()
}

// moverProcesoExecABlocked mueve un proceso de EXEC a BLOCKED
func (p *Service) moverProcesoExecABlocked(pid int) error {
	var proceso *internal.Proceso

	// Remover de EXEC usando función segura
	p.mutexExecQueue.Lock()
	for _, proc := range p.Planificador.ExecQueue {
		if proc.PCB.PID == pid {
			proceso = proc
			break
		}
	}

	if proceso != nil {
		var removido bool
		p.Planificador.ExecQueue, removido = p.removerDeCola(pid, p.Planificador.ExecQueue)
		if !removido {
			p.Log.Error("🚨Proceso no encontrado en ExecQueue durante dump_memory",
				log.IntAttr("pid", pid),
			)
		}

		if proceso.PCB.MetricasTiempo[internal.EstadoExec] != nil {
			proceso.PCB.MetricasTiempo[internal.EstadoExec].TiempoAcumulado +=
				time.Since(proceso.PCB.MetricasTiempo[internal.EstadoExec].TiempoInicio)
		}
	}

	if proceso == nil {
		p.mutexExecQueue.Unlock()
		return fmt.Errorf("proceso con PID %d no encontrado en EXEC", pid)
	}

	// Agregar a BLOCKED
	p.mutexBlockQueue.Lock()
	p.Planificador.BlockQueue = append(p.Planificador.BlockQueue, proceso)

	// Inicializar métricas de tiempo para BLOCKED
	if proceso.PCB.MetricasTiempo[internal.EstadoBloqueado] == nil {
		proceso.PCB.MetricasTiempo[internal.EstadoBloqueado] = &internal.EstadoTiempo{}
	}
	proceso.PCB.MetricasTiempo[internal.EstadoBloqueado].TiempoInicio = time.Now()
	proceso.PCB.MetricasEstado[internal.EstadoBloqueado]++

	p.Log.Info(fmt.Sprintf("## (%d) Pasa del estado EXEC al estado BLOCKED", proceso.PCB.PID))
	p.mutexBlockQueue.Unlock()
	p.mutexExecQueue.Unlock()

	// Notificar al planificador de mediano plazo
	p.CanalNuevoProcBlocked <- proceso

	return nil
}

// moverProcesoBlockedAReady mueve un proceso de BLOCKED a READY
func (p *Service) moverProcesoBlockedAReady(pid int) error {
	var proceso *internal.Proceso

	// Remover de BLOCKED usando función segura
	p.mutexBlockQueue.Lock()
	for _, proc := range p.Planificador.BlockQueue {
		if proc.PCB.PID == pid {
			proceso = proc
			break
		}
	}

	if proceso != nil {
		var removido bool
		p.Planificador.BlockQueue, removido = p.removerDeCola(pid, p.Planificador.BlockQueue)
		if !removido {
			p.Log.Error("🚨 Proceso no encontrado en BlockQueue durante moverProcesoBlockedAReady",
				log.IntAttr("pid", pid),
			)
		}
	}

	if proceso == nil {
		p.mutexBlockQueue.Unlock()
		return fmt.Errorf("proceso con PID %d no encontrado en BLOCKED", pid)
	}

	// Actualizar métricas de tiempo para BLOCKED
	if proceso.PCB.MetricasTiempo[internal.EstadoBloqueado] != nil {
		proceso.PCB.MetricasTiempo[internal.EstadoBloqueado].TiempoAcumulado +=
			time.Since(proceso.PCB.MetricasTiempo[internal.EstadoBloqueado].TiempoInicio)
	}

	// Agregar a READY
	p.mutexReadyQueue.Lock()
	p.Planificador.ReadyQueue = append(p.Planificador.ReadyQueue, proceso)
	// Inicializar métricas de tiempo para READY
	if proceso.PCB.MetricasTiempo[internal.EstadoReady] == nil {
		proceso.PCB.MetricasTiempo[internal.EstadoReady] = &internal.EstadoTiempo{}
	}
	proceso.PCB.MetricasTiempo[internal.EstadoReady].TiempoInicio = time.Now()
	proceso.PCB.MetricasEstado[internal.EstadoReady]++

	p.Log.Info(fmt.Sprintf("## (%d) Pasa del estado BLOCKED al estado READY", proceso.PCB.PID))
	p.mutexReadyQueue.Unlock()
	p.mutexBlockQueue.Unlock()

	p.canalNuevoProcesoReady <- struct{}{} // Notificar al planificador de corto plazo

	return nil
}

// moverProcesoBlockedAExit mueve un proceso de BLOCKED a EXIT
func (p *Service) moverProcesoBlockedAExit(pid int) error {
	var proceso *internal.Proceso

	// Remover de BLOCKED usando función segura
	p.mutexBlockQueue.Lock()
	for _, proc := range p.Planificador.BlockQueue {
		if proc.PCB.PID == pid {
			proceso = proc
			break
		}
	}

	// Eliminar usando
	if proceso != nil {
		var removido bool
		p.Planificador.BlockQueue, removido = p.removerDeCola(pid, p.Planificador.BlockQueue)
		if !removido {
			p.Log.Error("🚨 Proceso no encontrado en BlockQueue durante moverProcesoBlockedAExit",
				log.IntAttr("pid", pid),
			)
		}
	}

	if proceso == nil {
		p.mutexBlockQueue.Unlock()
		return fmt.Errorf("proceso con PID %d no encontrado en BLOCKED", pid)
	}

	// Actualizar métricas de tiempo para BLOCKED
	if proceso.PCB.MetricasTiempo[internal.EstadoBloqueado] == nil {
		proceso.PCB.MetricasTiempo[internal.EstadoBloqueado] = &internal.EstadoTiempo{
			TiempoAcumulado: 0,
		}
	}
	proceso.PCB.MetricasTiempo[internal.EstadoBloqueado].TiempoAcumulado +=
		time.Since(proceso.PCB.MetricasTiempo[internal.EstadoBloqueado].TiempoInicio)

	// Agregar a EXIT
	p.Planificador.ExitQueue = append(p.Planificador.ExitQueue, proceso)

	// Inicializar métricas de tiempo para EXIT
	if proceso.PCB.MetricasTiempo[internal.EstadoExit] == nil {
		proceso.PCB.MetricasTiempo[internal.EstadoExit] = &internal.EstadoTiempo{}
	}
	proceso.PCB.MetricasTiempo[internal.EstadoExit].TiempoInicio = time.Now()
	proceso.PCB.MetricasEstado[internal.EstadoExit]++

	p.Log.Info(fmt.Sprintf("## (%d) Pasa del estado BLOCKED al estado EXIT", proceso.PCB.PID))
	p.mutexBlockQueue.Unlock()

	return nil
}
