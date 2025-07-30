package planificadores

import (
	"fmt"
	"strings"
	"time"

	"github.com/sisoputnfrba/tp-golang/kernel/internal"
	"github.com/sisoputnfrba/tp-golang/utils/log"
)

func (p *Service) SuspenderProcesoBloqueado() {
	for {
		//La funcion espera que entre un proceso en la cola de blocked
		proceso := <-p.CanalNuevoProcBlocked

		go func() {
			//Busco tiempo de espera para pasar a SuspendedBlocked
			//del archivo de configuración
			tiempoEspera := p.MedianoPlazoConfig.SuspensionTime
			//Espero que pase el tiempo determinado
			time.Sleep(time.Duration(tiempoEspera) * time.Millisecond)

			//Si el proceso sigue bloqueado, lo suspendemos
			p.mutexBlockQueue.Lock()
			sigueBloqueado := estaEnCola(proceso, p.Planificador.BlockQueue)

			if sigueBloqueado {
				//Mover de blocked a suspended blocked
				p.removerDeCola(proceso.PCB.PID, p.Planificador.BlockQueue)

				p.mutexSuspBlockQueue.Lock()
				p.Planificador.SuspBlockQueue = append(p.Planificador.SuspBlockQueue, proceso)
				p.mutexSuspBlockQueue.Unlock()

				//Actualizar métricas
				tiempo := proceso.PCB.MetricasTiempo[internal.EstadoBloqueado]
				tiempo.TiempoAcumulado += time.Since(tiempo.TiempoInicio)

				if proceso.PCB.MetricasTiempo[internal.EstadoSuspBloqueado] == nil {
					proceso.PCB.MetricasTiempo[internal.EstadoSuspBloqueado] = &internal.EstadoTiempo{}
				}
				proceso.PCB.MetricasTiempo[internal.EstadoSuspBloqueado].TiempoInicio = time.Now()
				proceso.PCB.MetricasEstado[internal.EstadoSuspBloqueado]++

				//Log obligatorio: Cambio de estado
				// "## (<PID>) Pasa del estado <ESTADO_ANTERIOR> al estado <ESTADO_ACTUAL>"
				p.Log.Info(fmt.Sprintf("## (%d) Pasa del estado BLOCKED al estado SUSP.BLOCKED", proceso.PCB.PID))

				//Notificar a memoria que debe swappear
				go p.avisarAMemoriaSwap(proceso)

				//Intentar traer procesos desde SUSP READY o NEW a memoria
				p.CheckearEspacioEnMemoria()

			}
			p.mutexBlockQueue.Unlock()
		}()
	}
}

// ManejarFinIO Cuando un proceso en SUSP.BLOCKED finalizo su IO, deberá pasar a
// SUSP.READY y quedar a la espera de su oportunidad de pasar a READY.
func (p *Service) ManejarFinIO(proceso *internal.Proceso) {
	if proceso == nil {
		p.Log.Error("ManejarFinIO: proceso es nil")
		return
	}

	p.mutexSuspBlockQueue.Lock()
	estabaSuspendido := estaEnCola(proceso, p.Planificador.SuspBlockQueue)
	if estabaSuspendido {
		var removido bool
		p.Planificador.SuspBlockQueue, removido = p.removerDeCola(proceso.PCB.PID, p.Planificador.SuspBlockQueue)
		if !removido {
			p.Log.Debug("🚨 Proceso no encontrado en SuspBlockQueue durante ManejarFinIO",
				log.IntAttr("pid", proceso.PCB.PID),
			)
		}

		p.mutexSuspReadyQueue.Lock()
		p.Planificador.SuspReadyQueue = append(p.Planificador.SuspReadyQueue, proceso)
		p.mutexSuspReadyQueue.Unlock()
		//Notifico que hay un nuevo proceso en SuspReady?
		//p.CanalNewProcSuspReady <- proceso

		//Actualizar métricas
		tiempo := proceso.PCB.MetricasTiempo[internal.EstadoSuspBloqueado]
		tiempo.TiempoAcumulado += time.Since(tiempo.TiempoInicio)

		if proceso.PCB.MetricasTiempo[internal.EstadoSuspReady] == nil {
			proceso.PCB.MetricasTiempo[internal.EstadoSuspReady] = &internal.EstadoTiempo{}
		}
		proceso.PCB.MetricasTiempo[internal.EstadoSuspReady].TiempoInicio = time.Now()
		proceso.PCB.MetricasEstado[internal.EstadoSuspReady]++

		//Log obligatorio: Cambio de estado
		// "## (<PID>) Pasa del estado <ESTADO_ANTERIOR> al estado <ESTADO_ACTUAL>"
		p.Log.Info(fmt.Sprintf("## (%d) Pasa del estado SUSP.BLOCKED al estado SUSP.READY", proceso.PCB.PID))

		// Checkear si hay espacio en memoria para traer procesos suspendidos
		p.CheckearEspacioEnMemoria()

	} else {
		//Proceso estaba en BLOCKED → READY
		//Esto no se si tiene que estar aca, puede ser logica repetida
		p.mutexBlockQueue.Lock()
		var removido bool
		p.Planificador.BlockQueue, removido = p.removerDeCola(proceso.PCB.PID, p.Planificador.BlockQueue)
		if !removido {
			p.Log.Debug("🚨 Proceso no encontrado en BlockQueue durante ManejarFinIO",
				log.IntAttr("pid", proceso.PCB.PID),
			)
		}
		if proceso.PCB.MetricasTiempo[internal.EstadoBloqueado] == nil {
			proceso.PCB.MetricasTiempo[internal.EstadoBloqueado] = &internal.EstadoTiempo{
				TiempoAcumulado: 0,
			}
		}
		proceso.PCB.MetricasTiempo[internal.EstadoBloqueado].TiempoAcumulado += time.Since(proceso.PCB.MetricasTiempo[internal.EstadoBloqueado].TiempoInicio)
		p.mutexBlockQueue.Unlock()

		p.mutexReadyQueue.Lock()
		p.Planificador.ReadyQueue = append(p.Planificador.ReadyQueue, proceso)
		if proceso.PCB.MetricasTiempo[internal.EstadoReady] == nil {
			proceso.PCB.MetricasTiempo[internal.EstadoReady] = &internal.EstadoTiempo{}
		}
		proceso.PCB.MetricasTiempo[internal.EstadoReady].TiempoInicio = time.Now()
		proceso.PCB.MetricasEstado[internal.EstadoReady]++
		p.mutexReadyQueue.Unlock()

		//Log obligatorio: Cambio de estado
		// "## (<PID>) Pasa del estado <ESTADO_ANTERIOR> al estado <ESTADO_ACTUAL>"
		p.Log.Info(fmt.Sprintf("## (%d) Pasa del estado BLOCKED al estado READY", proceso.PCB.PID))

		// Notificar planificador corto plazo
		p.canalNuevoProcesoReady <- struct{}{}
	}
	p.mutexSuspBlockQueue.Unlock()
}

func estaEnCola(p *internal.Proceso, cola []*internal.Proceso) bool {
	if p == nil || p.PCB == nil {
		return false
	}
	if len(cola) == 0 {
		return false
	}

	for _, proc := range cola {
		if proc.PCB.PID == p.PCB.PID {
			return true
		}
	}
	return false
}

// avisarAMemoriaSwap notifica a memoria que debe realizar el swap del proceso
func (p *Service) avisarAMemoriaSwap(proceso *internal.Proceso) {
	err := p.Memoria.SwapProceso(proceso.PCB.PID)
	if err != nil {
		p.Log.Error("Error al notificar a memoria para swappear proceso",
			log.ErrAttr(err),
			log.IntAttr("pid", proceso.PCB.PID),
		)
		return
	}
}

func (p *Service) BuscarProcesoEnCola(pid int, cola string) *internal.Proceso {
	colaString := strings.ToLower(cola)
	switch colaString {
	case "exec":
		p.mutexExecQueue.RLock()
		for _, proc := range p.Planificador.ExecQueue {
			if proc.PCB.PID == pid {
				p.mutexExecQueue.RUnlock()
				return proc
			}
		}
		p.mutexExecQueue.RUnlock()
	case "suspended_blocked":
		p.mutexSuspBlockQueue.RLock()
		for _, proc := range p.Planificador.SuspBlockQueue {
			if proc.PCB.PID == pid {
				p.mutexSuspBlockQueue.RUnlock()
				return proc
			}
		}
		p.mutexSuspBlockQueue.RUnlock()
	case "blocked":
		p.mutexBlockQueue.RLock()
		for _, proc := range p.Planificador.BlockQueue {
			if proc.PCB.PID == pid {
				p.mutexBlockQueue.RUnlock()
				return proc
			}
		}
		p.mutexBlockQueue.RUnlock()
	case "ready":
		p.mutexReadyQueue.RLock()
		for _, proc := range p.Planificador.ReadyQueue {
			if proc.PCB.PID == pid {
				p.mutexReadyQueue.RUnlock()
				return proc
			}
		}
		p.mutexReadyQueue.RUnlock()
	case "suspended_ready":
		p.mutexSuspReadyQueue.RLock()
		for _, proc := range p.Planificador.SuspReadyQueue {
			if proc.PCB.PID == pid {
				p.mutexSuspReadyQueue.RUnlock()
				return proc
			}
		}
		p.mutexSuspReadyQueue.RUnlock()
	default:
		// Si no se especifica cola o es desconocida, buscar en todas las colas relevantes
		// Primero en BLOCKED (más probable para procesos de IO)
		p.mutexBlockQueue.RLock()
		for _, proc := range p.Planificador.BlockQueue {
			if proc.PCB.PID == pid {
				p.mutexBlockQueue.RUnlock()
				return proc
			}
		}
		p.mutexBlockQueue.RUnlock()

		// Luego en SUSP.BLOCKED
		p.mutexSuspBlockQueue.RLock()
		for _, proc := range p.Planificador.SuspBlockQueue {
			if proc.PCB.PID == pid {
				p.mutexSuspBlockQueue.RUnlock()
				return proc
			}
		}
		p.mutexSuspBlockQueue.RUnlock()
	}

	p.Log.Debug("Proceso no encontrado en la cola especificada",
		log.IntAttr("PID", pid),
		log.StringAttr("Cola", cola),
	)
	return nil
}
