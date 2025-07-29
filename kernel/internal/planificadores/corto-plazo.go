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
		// Esperar hasta que haya trabajo que hacer
		select {
		case <-p.canalNuevoProcesoReady:
			// Hay nuevos procesos en Ready, procesarlos
			p.Log.Debug("Notificación de nuevo proceso en Ready recibida")
		default:
			// No hay notificaciones pendientes, continuar
		}

		// Procesar todos los procesos en ReadyQueue
		for len(p.Planificador.ReadyQueue) > 0 {
			// Usar versión bloqueante para adquirir CPU
			cpuLibre := p.BuscarCPUDisponible()

			if cpuLibre != nil {
				// Mover proceso de READY a EXEC de forma atómica
				p.mutexReadyQueue.Lock()
				procesoElegido := p.Planificador.ReadyQueue[0]
				p.Planificador.ReadyQueue = p.Planificador.ReadyQueue[1:]

				// Actualizar métricas de tiempo
				if procesoElegido.PCB.MetricasTiempo[internal.EstadoReady] != nil {
					procesoElegido.PCB.MetricasTiempo[internal.EstadoReady].TiempoAcumulado +=
						time.Since(procesoElegido.PCB.MetricasTiempo[internal.EstadoReady].TiempoInicio)
				}

				// Crear una copia del proceso para la goroutine
				procesoCopia := *procesoElegido

				p.mutexReadyQueue.Unlock()

				// Agregar inmediatamente a ExecQueue de forma síncrona
				p.mutexExecQueue.Lock()
				p.Planificador.ExecQueue = append(p.Planificador.ExecQueue, procesoElegido)

				if procesoElegido.PCB.MetricasTiempo[internal.EstadoExec] == nil {
					procesoElegido.PCB.MetricasTiempo[internal.EstadoExec] = &internal.EstadoTiempo{}
				}
				procesoElegido.PCB.MetricasTiempo[internal.EstadoExec].TiempoInicio = time.Now()
				procesoElegido.PCB.MetricasEstado[internal.EstadoExec]++

				//Log obligatorio: Cambio de estado
				p.Log.Info(fmt.Sprintf("## (%d) Pasa del estado READY al estado EXEC", procesoElegido.PCB.PID))

				// Configurar CPU con los valores copiados
				cpuLibre.Proceso.PC = procesoCopia.PCB.PC
				cpuLibre.Proceso.PID = procesoCopia.PCB.PID

				// Ejecutar proceso en rutina para permitir concurrencia
				go func(cpuElegida *cpu.Cpu, proceso *internal.Proceso) {

					p.Log.Debug("CPU seleccionada para proceso",
						log.StringAttr("cpu_id", cpuElegida.ID),
						log.IntAttr("pid", proceso.PCB.PID),
					)

					newPC, _, _ := cpuElegida.DispatchProcess()
					if proceso != nil && proceso.PCB != nil {
						proceso.PCB.PC = newPC
					}

					// Liberar CPU usando semáforo
					p.LiberarCPU(cpuElegida)

					p.Log.Debug("Proceso completado en CPU",
						log.StringAttr("cpu_id", cpuElegida.ID),
						//log.IntAttr("pid", proceso.PCB.PID),
						log.IntAttr("pc_final", newPC),
					)

				}(cpuLibre, procesoElegido)
				p.mutexExecQueue.Unlock()
			} else {
				// No hay CPUs libres, salir del bucle interno
				p.Log.Debug("No hay CPUs libres, esperando...")
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
		// Esperar por notificación de nuevo proceso o procesar si ya hay procesos
		select {
		case <-p.canalNuevoProcesoReady:
			p.Log.Debug("Notificación de nuevo proceso en Ready recibida (SJF)")
		default:
			// No hay notificación, pero verificar si hay procesos para procesar
			if len(p.Planificador.ReadyQueue) == 0 {
				// No hay procesos, esperar por notificación
				p.Log.Debug("No hay procesos en ReadyQueue, esperando notificación... (SJF)")
				<-p.canalNuevoProcesoReady
			}
		}

		// Procesar todos los procesos en ReadyQueue
		for len(p.Planificador.ReadyQueue) > 0 {
			p.mutexReadyQueue.Lock()
			p.ordenarColaReadySjf() // Ordena la cola de ReadyQueue por ráfaga estimada

			procesoNuevo := p.Planificador.ReadyQueue[0]
			p.Planificador.ReadyQueue = p.Planificador.ReadyQueue[1:]
			p.mutexReadyQueue.Unlock()

			// Evaluar desalojo (independientemente de CPUs libres)
			procesoADesalojar := p.evaluarDesalojo(procesoNuevo)
			if procesoADesalojar != nil {
				cpuLiberada := p.desalojarProceso(procesoADesalojar)

				// Log obligatorio: Desalojo de SJF/SRT
				//"## (<PID>) - Desalojado por algoritmo SJF/SRT"
				p.Log.Info(fmt.Sprintf("## (%d) - Desalojado por algoritmo SJF/SRT", procesoADesalojar.PCB.PID))

				// Después del desalojo, asignar el nuevo proceso
				p.asignarProcesoACPU(procesoNuevo, cpuLiberada)
			} else {
				// Si no hay desalojo, buscar CPU libre
				cpuLibre := p.IntentarBuscarCPUDisponible()

				if cpuLibre != nil {
					// Hay CPU libre, asignar el proceso con ráfaga más corta
					p.asignarProcesoACPU(procesoNuevo, cpuLibre)
				} else {
					// No hay CPUs libres y no se puede desalojar, reencolar y esperar
					p.Log.Debug("No se puede desalojar ningún proceso con SRT y no hay CPUs libres, reencolando proceso...")
					// Volver a poner el proceso en ReadyQueue para reintentar más tarde
					p.mutexReadyQueue.Lock()
					p.Planificador.ReadyQueue = append([]*internal.Proceso{procesoNuevo}, p.Planificador.ReadyQueue...)
					p.mutexReadyQueue.Unlock()
					p.canalNuevoProcesoReady <- struct{}{} // Notificar que hay un nuevo proceso en ReadyQueue
					break                                  // Salir del bucle y esperar por el próximo evento
				}
			}
		}
	}
}

// PlanificarCortoPlazoSjfSinDesalojo planifica los procesos de corto plazo utilizando el algoritmo SJF sin desalojo.
func (p *Service) PlanificarCortoPlazoSjfSinDesalojo() {
	for {
		// Esperar por notificación de nuevo proceso o procesar si ya hay procesos
		select {
		case <-p.canalNuevoProcesoReady:
			p.Log.Debug("Notificación de nuevo proceso en Ready recibida (SJF)")
		default:
			// No hay notificación, pero verificar si hay procesos para procesar
			if len(p.Planificador.ReadyQueue) == 0 {
				// No hay procesos, esperar por notificación
				p.Log.Debug("No hay procesos en ReadyQueue, esperando notificación... (SJF)")
				<-p.canalNuevoProcesoReady
			}
		}

		// Procesar todos los procesos en ReadyQueue
		for len(p.Planificador.ReadyQueue) > 0 {
			p.mutexReadyQueue.Lock()

			p.ordenarColaReadySjf()
			// Usar versión bloqueante para adquirir CPU
			cpuLibre := p.BuscarCPUDisponible()

			if cpuLibre != nil {
				// Asignar el proceso con ráfaga más corta (el primero de ReadyQueue)
				procesoMasCorto := p.Planificador.ReadyQueue[0]
				p.Planificador.ReadyQueue = p.Planificador.ReadyQueue[1:]
				//p.mutexReadyQueue.Unlock()

				p.asignarProcesoACPU(procesoMasCorto, cpuLibre)
			}
			p.mutexReadyQueue.Unlock()
		}
	}
}

// ordenarColaReadySjf ordena la cola de ReadyQueue por ráfaga estimada ascendente
// Si dos procesos tienen la misma ráfaga, el proceso más nuevo (mayor PID) va después
func (p *Service) ordenarColaReadySjf() {
	sort.SliceStable(p.Planificador.ReadyQueue, func(i, j int) bool {
		estI := p.Planificador.ReadyQueue[i].EstimacionRafaga
		estJ := p.Planificador.ReadyQueue[j].EstimacionRafaga

		if estI < estJ {
			return true
		}
		if estI > estJ {
			return false
		}
		// Si la ráfaga es igual, el proceso con menor PID (más viejo) va primero
		return p.Planificador.ReadyQueue[i].PCB.PID < p.Planificador.ReadyQueue[j].PCB.PID
	})

	p.Log.Debug("Cola ReadyQueue ordenada por ráfaga estimada (SJF)",
		log.AnyAttr("procesos_en_ready", p.Planificador.ReadyQueue))
}

func estimacionRestante(p *internal.Proceso) float64 {
	tiempoEnExec := time.Since(p.InstanteInicio).Milliseconds()

	if float64(tiempoEnExec) >= p.EstimacionRafaga {
		return 0
	}

	return p.EstimacionRafaga - float64(tiempoEnExec)
}

func (p *Service) recalcularRafaga(proceso *internal.Proceso, rafagaReal float64) {
	alpha := p.SjfConfig.Alpha
	proceso.UltimaRafagaEstimada = proceso.EstimacionRafaga
	proceso.EstimacionRafaga = alpha*rafagaReal + (1-alpha)*proceso.UltimaRafagaEstimada
}

func (p *Service) procesoADesalojar(nuevaEstimacion float64) int {
	maxTiempoRestante := -1.0
	indiceProceso := -1

	for i, proceso := range p.Planificador.ExecQueue {
		// Tiempo restante estimado
		tiempoRestante := estimacionRestante(proceso)

		if tiempoRestante > nuevaEstimacion && tiempoRestante > maxTiempoRestante {
			maxTiempoRestante = tiempoRestante
			indiceProceso = i
		}
	}

	return indiceProceso
}

// evaluarDesalojo evalúa si el proceso nuevo debe desalojar algún proceso en ejecución
func (p *Service) evaluarDesalojo(procesoNuevo *internal.Proceso) *internal.Proceso {
	if len(p.Planificador.ExecQueue) == 0 {
		p.Log.Debug("No hay procesos en ExecQueue para evaluar desalojo")
		return nil
	}

	p.mutexExecQueue.Lock()

	index := p.procesoADesalojar(procesoNuevo.EstimacionRafaga)
	if index == -1 {
		p.Log.Debug("No se encontró proceso para desalojar")
		p.mutexExecQueue.Unlock()
		return nil
	}

	// Verificar que el índice sea válido
	if index < 0 || index >= len(p.Planificador.ExecQueue) {
		p.Log.Error("Índice de proceso a desalojar fuera de rango",
			log.IntAttr("index", index),
			log.IntAttr("total_procesos_exec", len(p.Planificador.ExecQueue)),
		)
		p.mutexExecQueue.Unlock()
		return nil
	}

	procesoADesalojar := p.Planificador.ExecQueue[index]

	p.mutexExecQueue.Unlock()

	return procesoADesalojar
}

// actualizarRafagaAnterior actualiza la ráfaga anterior y estimación anterior del proceso.
// Debe llamarse cada vez que un proceso deja de ejecutarse (por desalojo, IO, finalización, etc.)
// Calcula la ráfaga estimada usando la fórmula: Est(n+1) = α * R(n) + (1-α) * Est(n) donde:
//
// - Est(n)=Estimado de la ráfaga anterior
// - R(n) = Lo que realmente ejecutó de la ráfaga anterior en la CPU
// - Est(n+1) = El estimado de la próxima ráfaga
func (p *Service) actualizarRafagaAnterior(proceso *internal.Proceso, rafagaReal int64) {
	alpha := p.SjfConfig.Alpha
	proceso.UltimaRafagaReal = float64(rafagaReal) // Guardar la ráfaga real en milisegundos
	proceso.UltimaRafagaEstimada = proceso.EstimacionRafaga
	proceso.EstimacionRafaga = alpha*float64(rafagaReal) + (1-alpha)*proceso.UltimaRafagaEstimada
}

// desalojarProceso desaloja un proceso de la CPU y lo devuelve a ReadyQueue
func (p *Service) desalojarProceso(proceso *internal.Proceso) *cpu.Cpu {
	p.mutexExecQueue.Lock()

	cpuFound := p.buscarCPUPorPID(proceso.PCB.PID)
	if cpuFound == nil {
		p.Log.Error("No se encontró CPU para el proceso a desalojar",
			log.IntAttr("pid", proceso.PCB.PID),
		)
		p.mutexExecQueue.Unlock()
		return nil
	}

	p.Log.Debug("CPU encontrada para desalojo",
		log.IntAttr("pid", proceso.PCB.PID),
		log.StringAttr("cpu_id", cpuFound.ID),
	)

	// Enviar interrupción de desalojo
	enviado := cpuFound.EnviarInterrupcion("Desalojo", false)
	if !enviado {
		p.Log.Error("Error al enviar interrupción de desalojo",
			log.IntAttr("pid", proceso.PCB.PID))
		p.mutexExecQueue.Unlock()
		return nil
	}

	// Remover de ExecQueue con protección de mutex usando función segura
	var removido bool
	p.Planificador.ExecQueue, removido = p.removerDeCola(proceso.PCB.PID, p.Planificador.ExecQueue)
	if !removido {
		p.Log.Error("🚨 CRÍTICO: Proceso a desalojar no estaba en ExecQueue",
			log.IntAttr("pid", proceso.PCB.PID),
		)
	}

	// Actualizar métricas de Exec
	if proceso.PCB.MetricasTiempo[internal.EstadoExec] != nil {
		proceso.PCB.MetricasTiempo[internal.EstadoExec].TiempoAcumulado +=
			time.Since(proceso.PCB.MetricasTiempo[internal.EstadoExec].TiempoInicio)
	}

	p.mutexExecQueue.Unlock()

	// Devolver a ReadyQueue con protección de mutex
	p.mutexReadyQueue.Lock()

	// VERIFICAR: Asegurar que no esté ya en ReadyQueue (evitar duplicados)
	yaEnReady := false
	for _, proc := range p.Planificador.ReadyQueue {
		if proc.PCB.PID == proceso.PCB.PID {
			yaEnReady = true
			p.Log.Error("🚨 Proceso ya estaba en ReadyQueue durante desalojo",
				log.IntAttr("pid", proceso.PCB.PID),
			)
			break
		}
	}

	if !yaEnReady {
		p.Planificador.ReadyQueue = append(p.Planificador.ReadyQueue, proceso)
	}

	// Actualizar métricas de Ready
	if proceso.PCB.MetricasTiempo[internal.EstadoReady] == nil {
		proceso.PCB.MetricasTiempo[internal.EstadoReady] = &internal.EstadoTiempo{}
	}
	proceso.PCB.MetricasTiempo[internal.EstadoReady].TiempoInicio = time.Now()
	proceso.PCB.MetricasEstado[internal.EstadoReady]++

	p.mutexReadyQueue.Unlock()

	//Log obligatorio: Cambio de estado
	// "## (<PID>) Pasa del estado <ESTADO_ANTERIOR> al estado <ESTADO_ACTUAL>"
	p.Log.Info(fmt.Sprintf("## (%d) Pasa del estado EXEC al estado READY", proceso.PCB.PID))

	p.Log.Debug("Proceso desalojado por SRT",
		log.IntAttr("PID", proceso.PCB.PID),
	)

	// Notificar que hay un nuevo proceso en ReadyQueue
	p.canalNuevoProcesoReady <- struct{}{}

	return cpuFound
}

// buscarCPUPorPID busca una CPU que esté ejecutando un proceso específico por su PID
func (p *Service) buscarCPUPorPID(pid int) *cpu.Cpu {
	p.mutexCPUsConectadas.RLock()
	defer p.mutexCPUsConectadas.RUnlock()

	for _, cpuMatch := range p.CPUsConectadas {
		if cpuMatch.Proceso.PID == pid {
			p.Log.Debug("CPU encontrada para PID",
				log.IntAttr("pid", pid),
				log.StringAttr("cpu_id", cpuMatch.ID),
				log.AnyAttr("cpu_estado", cpuMatch.Estado),
			)
			return cpuMatch
		}
	}

	p.Log.Debug("CPU NO encontrada para PID",
		log.IntAttr("pid", pid),
		log.IntAttr("total_cpus", len(p.CPUsConectadas)),
	)
	return nil
}

// asignarProcesoACPU asigna un proceso a una CPU específica
func (p *Service) asignarProcesoACPU(proceso *internal.Proceso, cpuAsignada *cpu.Cpu) {
	// Actualizar la CPU con el proceso
	// Ejecutar en goroutine para no bloquear el planificador

	if cpuAsignada == nil || proceso.PCB == nil {
		p.Log.Error("CPU o proceso inválido al asignar a CPU",
			log.AnyAttr("cpu", cpuAsignada),
			log.AnyAttr("proceso", proceso),
		)
		return
	}

	// Actualizar métricas de tiempo
	if proceso.PCB.MetricasTiempo[internal.EstadoReady] != nil {
		proceso.PCB.MetricasTiempo[internal.EstadoReady].TiempoAcumulado +=
			time.Since(proceso.PCB.MetricasTiempo[internal.EstadoReady].TiempoInicio)
	}

	// IMPORTANTE: Usar orden consistente de mutex (ExecQueue -> CPUsConectadas) para evitar deadlocks
	p.mutexExecQueue.Lock()
	p.mutexCPUsConectadas.Lock()

	// Agregar a ExecQueue primero
	p.Planificador.ExecQueue = append(p.Planificador.ExecQueue, proceso)

	if proceso.PCB.MetricasTiempo[internal.EstadoExec] == nil {
		proceso.PCB.MetricasTiempo[internal.EstadoExec] = &internal.EstadoTiempo{}
	}
	proceso.PCB.MetricasTiempo[internal.EstadoExec].TiempoInicio = time.Now()
	proceso.PCB.MetricasEstado[internal.EstadoExec]++

	// Actualizar la CPU con el proceso
	cpuAsignada.Proceso.PID = proceso.PCB.PID
	cpuAsignada.Proceso.PC = proceso.PCB.PC

	// Liberar mutex en orden inverso (LIFO) para evitar deadlocks
	p.mutexCPUsConectadas.Unlock()
	p.mutexExecQueue.Unlock()

	//Log obligatorio: Cambio de estado
	p.Log.Info(fmt.Sprintf("## (%d) Pasa del estado READY al estado EXEC", proceso.PCB.PID))

	p.Log.Debug("Proceso asignado a CPU con SRT/SJF",
		log.IntAttr("PID", proceso.PCB.PID),
		log.StringAttr("CPU_ID", cpuAsignada.ID),
	)

	// Ejecutar en goroutine para permitir concurrencia
	go func(cpuElegida *cpu.Cpu, procesoExec *internal.Proceso) {
		defer func() {
			// Asegurar que la CPU siempre se libere
			p.LiberarCPU(cpuElegida)
		}()

		proceso.InstanteInicio = time.Now()
		// Enviar proceso a la CPU
		newPC, motivo, rafaga := cpuElegida.DispatchProcess()
		if procesoExec != nil && procesoExec.PCB != nil {
			procesoExec.PCB.PC = newPC
		}

		// Actualizar ráfaga anterior y estimación
		p.actualizarRafagaAnterior(procesoExec, rafaga)

		// Si hubo error al ejecutar el ciclo, registrar pero no remover de ExecQueue
		// La remoción se manejará por el ciclo principal del planificador
		if motivo != "Proceso ejecutado exitosamente" {
			p.Log.Debug("Error al ejecutar proceso en CPU",
				log.IntAttr("PID", procesoExec.PCB.PID),
				log.StringAttr("motivo", motivo),
			)
		}
	}(cpuAsignada, proceso)
}
