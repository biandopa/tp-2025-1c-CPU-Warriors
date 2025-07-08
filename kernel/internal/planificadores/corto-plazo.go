package planificadores

import (
	"fmt"
	"sort"
	"time"

	"github.com/sisoputnfrba/tp-golang/kernel/internal"
	"github.com/sisoputnfrba/tp-golang/kernel/pkg/cpu"
	"github.com/sisoputnfrba/tp-golang/utils/log"
)

func (p *Service) PlanificadorCortoPlazo() {
	switch p.ShortTermAlgorithm {
	case "FIFO":
		go p.PlanificadorCortoPlazoFIFO()
	case "SJF":
		go p.PlanificarCortoPlazoSjfSinDesalojo()
	case "SRT":
		go p.PlanificarCortoPlazoSjfDesalojo()
	default:
		p.Log.Warn("Algoritmo de corto plazo no reconocido")
	}
}

func (p *Service) PlanificadorCortoPlazoFIFO() {
	for {
		// Procesar todos los procesos en ReadyQueue
		for len(p.Planificador.ReadyQueue) > 0 {
			// Usar semáforo para adquirir CPU (bloqueante)
			cpuLibre := p.BuscarCPUDisponible()

			if cpuLibre != nil {
				// Mover proceso de READY a EXEC
				p.mutexReadyQueue.Lock()
				procesoElegido := p.Planificador.ReadyQueue[0]
				p.Planificador.ReadyQueue = p.Planificador.ReadyQueue[1:]
				p.mutexReadyQueue.Unlock()

				// Ejecutar proceso en rutina para permitir concurrencia
				go func(cpuElegida *cpu.Cpu, proceso *internal.Proceso) {
					// Actualizar métricas de tiempo
					if proceso.PCB.MetricasTiempo[internal.EstadoReady] != nil {
						proceso.PCB.MetricasTiempo[internal.EstadoReady].TiempoAcumulado += time.Since(proceso.PCB.MetricasTiempo[internal.EstadoReady].TiempoInicio)
					}

					p.mutexExecQueue.Lock()
					p.Planificador.ExecQueue = append(p.Planificador.ExecQueue, proceso)
					p.mutexExecQueue.Unlock()

					if proceso.PCB.MetricasTiempo[internal.EstadoExec] == nil {
						proceso.PCB.MetricasTiempo[internal.EstadoExec] = &internal.EstadoTiempo{}
					}
					proceso.PCB.MetricasTiempo[internal.EstadoExec].TiempoInicio = time.Now()
					proceso.PCB.MetricasEstado[internal.EstadoExec]++

					//Log obligatorio: Cambio de estado
					// “## (<PID>) Pasa del estado <ESTADO_ANTERIOR> al estado <ESTADO_ACTUAL>”
					p.Log.Info(fmt.Sprintf("## (%d) Pasa del estado READY al estado EXEC", proceso.PCB.PID))

					// Usar los valores copiados
					cpuElegida.Proceso.PC = procesoElegido.PCB.PC
					cpuElegida.Proceso.PID = procesoElegido.PCB.PID

					p.Log.Debug("CPU seleccionada para proceso",
						log.StringAttr("cpu_id", cpuElegida.ID),
						log.IntAttr("pid", proceso.PCB.PID),
					)

					newPC, _ := cpuElegida.DispatchProcess()
					proceso.PCB.PC = newPC

					// Liberar CPU usando semáforo
					p.LiberarCPU(cpuElegida)

					p.Log.Debug("Proceso completado en CPU",
						log.StringAttr("cpu_id", cpuElegida.ID),
						log.IntAttr("pid", proceso.PCB.PID),
						log.IntAttr("pc_final", newPC),
					)
				}(cpuLibre, procesoElegido)
			} else {
				// No hay CPUs libres, salir del bucle
				p.Log.Debug("No hay CPUs libres, saliendo del bucle")

				break
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
		p.odenarColaReadySjf() // Ordena la cola de ReadyQueue por ráfaga estimada

		// Procesar todos los procesos en ReadyQueue
		for len(p.Planificador.ReadyQueue) > 0 {
			// Intentar adquirir CPU libre sin bloquear
			cpuLibre := p.IntentarBuscarCPUDisponible()

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

					// Log obligatorio: Desalojo de SJF/SRT
					//“## (<PID>) - Desalojado por algoritmo SJF/SRT”
					p.Log.Info(fmt.Sprintf("## (%d) - Desalojado por algoritmo SJF/SRT", procesoADesalojar.PCB.PID))

					// Después del desalojo, asignar el nuevo proceso
					cpuLiberada := p.buscarCPUPorPID(procesoADesalojar.PCB.PID)
					if cpuLiberada != nil {
						p.asignarProcesoACPU(procesoNuevo, cpuLiberada)
					}

					time.Sleep(100 * time.Millisecond) // Esperar un poco antes de continuar
				}
			}
		}
	}
}

// PlanificarCortoPlazoSjfSinDesalojo planifica los procesos de corto plazo utilizando el algoritmo SJF sin desalojo.
func (p *Service) PlanificarCortoPlazoSjfSinDesalojo() {
	for {
		p.odenarColaReadySjf()

		// Procesar todos los procesos en ReadyQueue
		for len(p.Planificador.ReadyQueue) > 0 {
			// Usar semáforo para adquirir CPU (bloqueante)
			if cpuLibre := p.BuscarCPUDisponible(); cpuLibre != nil {
				// Asignar el proceso con ráfaga más corta (el primero de ReadyQueue)
				procesoMasCorto := p.Planificador.ReadyQueue[0]

				p.asignarProcesoACPU(procesoMasCorto, cpuLibre)
			}
		}
	}
}

// odenarColaReadySfj ordena la cola de ReadyQueue por ráfaga estimada ascendente
func (p *Service) odenarColaReadySjf() {
	p.mutexReadyQueue.Lock()
	defer p.mutexReadyQueue.Unlock()

	sort.Slice(p.Planificador.ReadyQueue, func(i, j int) bool {
		return p.calcularRafagaEstimada(p.Planificador.ReadyQueue[i]) < p.calcularRafagaEstimada(p.Planificador.ReadyQueue[j])
	})
}

// calcularRafagaEstimada calcula la ráfaga estimada usando la fórmula: Est(n+1) = α * R(n) + (1-α) * Est(n)
// donde:
// * Est(n)=Estimado de la ráfaga anterior
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
	cpuFound := p.buscarCPUPorPID(proceso.PCB.PID)
	if cpuFound != nil {
		cpuFound.EnviarInterrupcion("Desalojo", false)
		//p.LiberarCPU(cpuFound)
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

	//Log obligatorio: Cambio de estado
	// “## (<PID>) Pasa del estado <ESTADO_ANTERIOR> al estado <ESTADO_ACTUAL>”
	p.Log.Info(fmt.Sprintf("## (%d) Pasa del estado EXEC al estado READY", proceso.PCB.PID))

	// Actualizar métricas de Ready
	if proceso.PCB.MetricasTiempo[internal.EstadoReady] == nil {
		proceso.PCB.MetricasTiempo[internal.EstadoReady] = &internal.EstadoTiempo{}
	}
	proceso.PCB.MetricasTiempo[internal.EstadoReady].TiempoInicio = time.Now()
	proceso.PCB.MetricasEstado[internal.EstadoReady]++

	p.Log.Debug("Proceso desalojado por SJF",
		log.IntAttr("PID", proceso.PCB.PID),
	)
}

// buscarCPUPorPID busca una CPU que esté ejecutando un proceso específico por su PID
func (p *Service) buscarCPUPorPID(pid int) *cpu.Cpu {
	p.mutexCPUsConectadas.Lock()
	defer p.mutexCPUsConectadas.Unlock()

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

	// Usar orden consistente de locks para evitar deadlocks
	p.mutexReadyQueue.Lock()
	p.mutexExecQueue.Lock()

	// Remover el proceso de ReadyQueue
	p.Planificador.ReadyQueue = p.Planificador.ReadyQueue[1:]

	// Agregar a ExecQueue
	p.Planificador.ExecQueue = append(p.Planificador.ExecQueue, proceso)

	p.mutexExecQueue.Unlock()
	p.mutexReadyQueue.Unlock()

	if proceso.PCB.MetricasTiempo[internal.EstadoExec] == nil {
		proceso.PCB.MetricasTiempo[internal.EstadoExec] = &internal.EstadoTiempo{}
	}
	proceso.PCB.MetricasTiempo[internal.EstadoExec].TiempoInicio = time.Now()
	proceso.PCB.MetricasEstado[internal.EstadoExec]++

	p.Log.Debug("Proceso asignado a CPU con SJF",
		log.IntAttr("PID", proceso.PCB.PID),
		log.StringAttr("CPU_ID", cpuAsignada.ID),
	)

	// Ejecutar en goroutine para no bloquear el planificador
	go func(cpuElegida *cpu.Cpu, procesoExec *internal.Proceso) {
		// Actualizar la CPU con el proceso
		cpuElegida.Proceso.PID = procesoExec.PCB.PID
		cpuElegida.Proceso.PC = procesoExec.PCB.PC

		// Enviar proceso a la CPU
		newPC, _ := cpuElegida.DispatchProcess()

		procesoExec.PCB.PC = newPC

		// Liberar CPU usando semáforo
		p.LiberarCPU(cpuElegida)
	}(cpuAsignada, proceso)
}
