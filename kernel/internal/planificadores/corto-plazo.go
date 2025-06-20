package planificadores

import (
	"fmt"
	"time"

	"github.com/sisoputnfrba/tp-golang/kernel/internal"
	"github.com/sisoputnfrba/tp-golang/kernel/pkg/cpu"
	"github.com/sisoputnfrba/tp-golang/utils/log"
)

func (p *Service) PlanificadorCortoPlazo() {
	switch p.ShortTermAlgorithm {
	case "FIFO":
		go p.PlanificadorCortoPlazoFIFO()
	case "SJFSD":
		go p.PlanificarCortoPlazoSjfDesalojo()
	case "SJFD":
		go p.PlanificarCortoPlazoSjfDesalojo()
	default:
		p.Log.Warn("Algoritmo de corto plazo no reconocido")
	}
}

func (p *Service) PlanificadorCortoPlazoFIFO() {
	for {
		proceso := <-p.canalNuevoProcesoReady // Espera una notificación
		// Ordenar la cola de ReadyQueue al recibir un nuevo proceso
		p.ordenarColaReadyFIFO(proceso)

		for len(p.Planificador.ReadyQueue) > 0 { // Procesa mientras haya elementos en ReadyQueue

			var cpuSeleccionada *cpu.Cpu
			for {
				if len(p.CPUsConectadas) > 0 {
					for i := range p.CPUsConectadas {
						if p.CPUsConectadas[i].Estado {
							// Mover proceso de READY a EXEC
							p.mutexReadyQueue.Lock()
							p.Planificador.ReadyQueue = p.Planificador.ReadyQueue[1:]
							p.mutexReadyQueue.Unlock()

							timeNew := proceso.PCB.MetricasTiempo[internal.EstadoReady]
							timeNew.TiempoAcumulado += time.Since(timeNew.TiempoInicio)

							p.mutexExecQueue.Lock()
							p.Planificador.ExecQueue = append(p.Planificador.ExecQueue, proceso)
							p.mutexExecQueue.Unlock()

							if proceso.PCB.MetricasTiempo[internal.EstadoExec] == nil {
								proceso.PCB.MetricasTiempo[internal.EstadoExec] = &internal.EstadoTiempo{}
							}
							proceso.PCB.MetricasTiempo[internal.EstadoExec].TiempoInicio = time.Now()
							proceso.PCB.MetricasEstado[internal.EstadoExec]++

							p.Log.Info("Proceso movido de READY a EXEC",
								log.IntAttr("PID", proceso.PCB.PID),
							)
							cpuSeleccionada = p.CPUsConectadas[i]
							cpuSeleccionada.Proceso.PC = proceso.PCB.PC
							cpuSeleccionada.Proceso.PID = proceso.PCB.PID
							p.CPUsConectadas[i].Estado = false
							fmt.Println("CPU seleccionada:", cpuSeleccionada)

							cpuSeleccionada.DispatchProcess()
							break
						}
					}
				}
				if cpuSeleccionada != nil {
					break
				}
			}
		}
	}
}

// PlanificarCortoPlazoSjfDesalojo elige al proceso que posea la ráfaga de CPU más corta. Al ingresar un proceso en
// la cola de Ready y no haber CPUs libres, se debe evaluar si dicho proceso tiene una rafaga más corta que los que
// se encuentran en ejecución. En caso de ser así, se debe informar al CPU que posea al Proceso con el tiempo restante
// más alto que debe desalojar al mismo para que pueda ser planificado el nuevo.
func (p *Service) PlanificarCortoPlazoSjfDesalojo() {
	for {
		// TODO: Preguntar mañana cómo se maneja el canal con múltiples nuevos procesos
		proceso := <-p.canalNuevoProcesoReady // Espera una notificación
		p.odenarColaReadySjf(proceso)

		// Procesar todos los procesos en ReadyQueue
		for len(p.Planificador.ReadyQueue) > 0 {
			// Buscar CPU libre
			cpuLibre := p.buscarCPULibre()

			if cpuLibre != nil {
				// Hay CPU libre, asignar el proceso con ráfaga más corta (el primero de ReadyQueue)
				procesoMasCorto := p.Planificador.ReadyQueue[0]

				p.asignarProcesoACPU(procesoMasCorto, cpuLibre)
			} else {
				// No hay CPUs libres, evaluar desalojo
				p.mutexReadyQueue.Lock()
				procesoNuevo := p.Planificador.ReadyQueue[0]
				p.mutexReadyQueue.Unlock()

				procesoADesalojar := p.evaluarDesalojo(procesoNuevo)
				if procesoADesalojar != nil {
					p.desalojarProceso(procesoADesalojar)
					// Después del desalojo, asignar el nuevo proceso
					cpuLiberada := p.buscarCPUPorPID(procesoADesalojar.PCB.PID)
					if cpuLiberada != nil {
						p.asignarProcesoACPU(procesoNuevo, cpuLiberada)
					}
				}

				break // Salir del bucle si no hay CPUs libres y no se puede desalojar
			}
		}
	}
}

func (p *Service) ordenarColaReadyFIFO(proceso *internal.Proceso) {
	p.mutexReadyQueue.Lock()
	p.Planificador.ReadyQueue = append([]*internal.Proceso{proceso}, p.Planificador.ReadyQueue...)
	p.mutexReadyQueue.Unlock()
}

// odenarColaReadySfj ordena la cola de ReadyQueue por ráfaga estimada ascendente
func (p *Service) odenarColaReadySjf(proceso *internal.Proceso) {
	p.mutexReadyQueue.Lock()
	defer p.mutexReadyQueue.Unlock()

	// Insertar el nuevo proceso en la posición correcta
	inserted := false
	for i, proc := range p.Planificador.ReadyQueue {
		if p.calcularRafagaEstimada(proceso) < p.calcularRafagaEstimada(proc) {
			p.Planificador.ReadyQueue = append(p.Planificador.ReadyQueue[:i], append([]*internal.Proceso{proceso}, p.Planificador.ReadyQueue[i:]...)...)
			inserted = true
			break
		}
	}
	if !inserted {
		p.Planificador.ReadyQueue = append(p.Planificador.ReadyQueue, proceso)
	}
}

// buscarCPULibre encuentra una CPU que esté disponible
func (p *Service) buscarCPULibre() *cpu.Cpu {
	for i := range p.CPUsConectadas {
		if p.CPUsConectadas[i].Estado {
			return p.CPUsConectadas[i]
		}
	}
	return nil
}

// calcularRafagaEstimada calcula la ráfaga estimada usando la fórmula: Est(n+1) = α * R(n) + (1-α) * Est(n)
// donde: * Est(n)=Estimado de la ráfaga anterior
// * R(n) = Lo que realmente ejecutó de la ráfaga anterior en la CPU
// * Est(n+1) = El estimado de la próxima ráfaga
func (p *Service) calcularRafagaEstimada(proceso *internal.Proceso) float64 {
	// Si es la primera vez que se ejecuta, usar estimación inicial
	if proceso.PCB.RafagaAnterior == nil {
		return float64(p.SjfConfig.InitialEstimate)
	}

	// Aplicar fórmula: Est(n+1) = α * R(n) + (1-α) * Est(n)
	alpha := p.SjfConfig.Alpha
	rafagaReal := float64(proceso.PCB.RafagaAnterior.Nanoseconds() / 1000000) // Convertir a ms
	estimacionAnterior := proceso.PCB.EstimacionAnterior

	nuevaEstimacion := alpha*rafagaReal + (1-alpha)*estimacionAnterior
	return nuevaEstimacion
}

// evaluarDesalojo evalúa si el proceso nuevo debe desalojar algún proceso en ejecución
func (p *Service) evaluarDesalojo(procesoNuevo *internal.Proceso) *internal.Proceso {
	if len(p.Planificador.ExecQueue) == 0 {
		return nil
	}

	rafagaNuevo := p.calcularRafagaEstimada(procesoNuevo)
	var procesoADesalojar *internal.Proceso
	tiempoRestanteMayor := float64(0)

	for _, procesoEjecutando := range p.Planificador.ExecQueue {
		// Calcular tiempo restante del proceso en ejecución
		tiempoEjecutado := float64(time.Since(procesoEjecutando.PCB.MetricasTiempo[internal.EstadoExec].TiempoInicio).Nanoseconds() / 1000000)
		rafagaEstimada := p.calcularRafagaEstimada(procesoEjecutando)
		tiempoRestante := rafagaEstimada - tiempoEjecutado

		// Si el proceso nuevo tiene ráfaga menor que el tiempo restante y es el mayor tiempo restante
		if rafagaNuevo < tiempoRestante && tiempoRestante > tiempoRestanteMayor {
			tiempoRestanteMayor = tiempoRestante
			procesoADesalojar = procesoEjecutando
		}
	}

	return procesoADesalojar
}

// desalojarProceso desaloja un proceso de la CPU y lo devuelve a ReadyQueue
func (p *Service) desalojarProceso(proceso *internal.Proceso) {
	// Encontrar y liberar la CPU
	for i := range p.CPUsConectadas {
		if !p.CPUsConectadas[i].Estado {
			// Verificar si esta CPU está ejecutando el proceso a desalojar
			// (Esta lógica puede necesitar mejorarse dependiendo de cómo se maneje la asignación)
			p.CPUsConectadas[i].Estado = true
			break
		}
	}

	// Actualizar métricas del proceso
	if proceso.PCB.MetricasTiempo[internal.EstadoExec] != nil {
		tiempoEjecutado := time.Since(proceso.PCB.MetricasTiempo[internal.EstadoExec].TiempoInicio)
		proceso.PCB.MetricasTiempo[internal.EstadoExec].TiempoAcumulado += tiempoEjecutado

		// Guardar información para próxima estimación
		if proceso.PCB.RafagaAnterior == nil {
			proceso.PCB.RafagaAnterior = &tiempoEjecutado
		} else {
			*proceso.PCB.RafagaAnterior = tiempoEjecutado
		}
		proceso.PCB.EstimacionAnterior = p.calcularRafagaEstimada(proceso)
	}

	// Remover de ExecQueue
	for i, proc := range p.Planificador.ExecQueue {
		if proc.PCB.PID == proceso.PCB.PID {
			p.Planificador.ExecQueue = append(p.Planificador.ExecQueue[:i],
				p.Planificador.ExecQueue[i+1:]...)
			break
		}
	}

	// Devolver a ReadyQueue
	p.Planificador.ReadyQueue = append(p.Planificador.ReadyQueue, proceso)

	// Actualizar métricas de Ready
	if proceso.PCB.MetricasTiempo[internal.EstadoReady] == nil {
		proceso.PCB.MetricasTiempo[internal.EstadoReady] = &internal.EstadoTiempo{}
	}
	proceso.PCB.MetricasTiempo[internal.EstadoReady].TiempoInicio = time.Now()
	proceso.PCB.MetricasEstado[internal.EstadoReady]++

	p.Log.Info("Proceso desalojado por SJF",
		log.IntAttr("PID", proceso.PCB.PID),
	)

	// Enviar interrupción a la CPU para desalojar el proceso
	// Esto dependerá de la implementación específica de tu sistema
	// Por ahora se asume que el cambio de estado es suficiente
}

// buscarCPUPorPID busca una CPU que esté ejecutando un proceso específico por su PID
func (p *Service) buscarCPUPorPID(pid int) *cpu.Cpu {
	for _, cpuMatch := range p.CPUsConectadas {
		if cpuMatch.Proceso.PID == pid {
			return cpuMatch
		}
	}
	return nil
}

// asignarProcesoACPU asigna un proceso a una CPU específica
func (p *Service) asignarProcesoACPU(proceso *internal.Proceso, cpuAsignada *cpu.Cpu) {
	// Mover proceso de READY a EXEC
	timeReady := proceso.PCB.MetricasTiempo[internal.EstadoReady]
	if timeReady != nil {
		timeReady.TiempoAcumulado += time.Since(timeReady.TiempoInicio)
	}

	p.mutexReadyQueue.Lock()
	p.mutexExecQueue.Lock()
	p.Planificador.ExecQueue = append(p.Planificador.ExecQueue, proceso)
	p.Planificador.ReadyQueue = p.Planificador.ReadyQueue[1:] // Remover el proceso de ReadyQueue
	p.mutexReadyQueue.Unlock()
	p.mutexExecQueue.Unlock()

	if proceso.PCB.MetricasTiempo[internal.EstadoExec] == nil {
		proceso.PCB.MetricasTiempo[internal.EstadoExec] = &internal.EstadoTiempo{}
	}
	proceso.PCB.MetricasTiempo[internal.EstadoExec].TiempoInicio = time.Now()
	proceso.PCB.MetricasEstado[internal.EstadoExec]++

	// Marcar CPU como ocupada
	cpuAsignada.Estado = false

	p.Log.Debug("Proceso asignado a CPU con SJF",
		log.IntAttr("PID", proceso.PCB.PID),
		log.StringAttr("CPU_ID", cpuAsignada.ID),
	)

	// Actualizar la CPU con el proceso
	cpuAsignada.Proceso.PID = proceso.PCB.PID
	cpuAsignada.Proceso.PC = proceso.PCB.PC

	// Enviar proceso a la CPU
	cpuAsignada.DispatchProcess()
}
