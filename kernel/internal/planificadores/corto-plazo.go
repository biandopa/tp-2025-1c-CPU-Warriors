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
				// Mover proceso de READY a EXEC
				p.mutexReadyQueue.Lock()
				procesoElegido := p.Planificador.ReadyQueue[0]
				p.Planificador.ReadyQueue = p.Planificador.ReadyQueue[1:]
				p.mutexReadyQueue.Unlock()

				// Ejecutar proceso en rutina para permitir concurrencia
				go func(cpuElegida *cpu.Cpu, proceso *internal.Proceso) {
					// Actualizar métricas de tiempo
					if proceso.PCB.MetricasTiempo[internal.EstadoReady] != nil {
						proceso.PCB.MetricasTiempo[internal.EstadoReady].TiempoAcumulado +=
							time.Since(proceso.PCB.MetricasTiempo[internal.EstadoReady].TiempoInicio)
					}

					p.mutexExecQueue.Lock()
					p.Planificador.ExecQueue = append(p.Planificador.ExecQueue, proceso)

					if proceso.PCB.MetricasTiempo[internal.EstadoExec] == nil {
						proceso.PCB.MetricasTiempo[internal.EstadoExec] = &internal.EstadoTiempo{}
					}
					proceso.PCB.MetricasTiempo[internal.EstadoExec].TiempoInicio = time.Now()
					proceso.PCB.MetricasEstado[internal.EstadoExec]++
					p.mutexExecQueue.Unlock()

					//Log obligatorio: Cambio de estado
					// "## (<PID>) Pasa del estado <ESTADO_ANTERIOR> al estado <ESTADO_ACTUAL>"
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
						//log.IntAttr("pid", proceso.PCB.PID),
						log.IntAttr("pc_final", newPC),
					)

				}(cpuLibre, procesoElegido)
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

		for {
			p.mutexReadyQueue.Lock()
			if len(p.Planificador.ReadyQueue) == 0 {
				p.mutexReadyQueue.Unlock()
				break
			}

			p.ordenarColaReadySjf() // Ordena la cola de ReadyQueue por ráfaga estimada

			procesoNuevo := p.Planificador.ReadyQueue[0]
			p.mutexReadyQueue.Unlock()

			// Evaluar desalojo
			procesoADesalojar := p.evaluarDesalojo(procesoNuevo)
			if procesoADesalojar != nil {
				// IMPORTANTE: NO remover de ReadyQueue hasta que CPU esté realmente disponible

				// Después del desalojo, ESPERAR hasta que CPU esté realmente disponible
				p.Log.Debug("Esperando CPU tras desalojo",
					log.IntAttr("pid_nuevo", procesoNuevo.PCB.PID),
					log.IntAttr("pid_desalojado", procesoADesalojar.PCB.PID),
				)

				cpuLibre := p.BuscarCPUDisponible()
				if cpuLibre != nil {
					p.Log.Debug("CPU disponible tras desalojo completado",
						log.StringAttr("cpu_id", cpuLibre.ID),
						log.IntAttr("pid_nuevo", procesoNuevo.PCB.PID),
					)
					p.asignarProcesoACPU(procesoNuevo, cpuLibre)
				} else {
					p.Log.Error("🚨 No se pudo obtener CPU tras desalojo - estado inconsistente")
					break
				}
			} else {
				// Si no hay desalojo, buscar CPU libre con timeout
				cpuLibre := p.IntentarBuscarCPUDisponible()
				if cpuLibre != nil {
					// Hay CPU libre, asignar el proceso con ráfaga más corta
					p.asignarProcesoACPU(procesoNuevo, cpuLibre)
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
		for {
			p.mutexReadyQueue.Lock()
			if len(p.Planificador.ReadyQueue) == 0 {
				p.mutexReadyQueue.Unlock()
				break // Salir del bucle si no hay más procesos en ReadyQueue
			}

			p.ordenarColaReadySjf()
			// Asignar el proceso con ráfaga más corta (el primero de ReadyQueue)
			procesoMasCorto := p.Planificador.ReadyQueue[0]
			p.mutexReadyQueue.Unlock()

			cpuLibre := p.BuscarCPUDisponible()
			if cpuLibre != nil {
				p.asignarProcesoACPU(procesoMasCorto, cpuLibre)
			}
		}
	}
}

// ordenarColaReadySfj ordena la cola de ReadyQueue por ráfaga estimada ascendente
func (p *Service) ordenarColaReadySjf() {
	sort.Slice(p.Planificador.ReadyQueue, func(i, j int) bool {
		return p.calcularSiguienteEstimacion(p.Planificador.ReadyQueue[i]) < p.calcularSiguienteEstimacion(p.Planificador.ReadyQueue[j])
	})
}

// calcularSiguienteEstimacion calcula la ráfaga estimada usando la fórmula: Est(n+1) = α * R(n) + (1-α) * Est(n)
// donde:
// * Est(n)=Estimado de la ráfaga anterior
// * R(n) = Lo que realmente ejecutó de la ráfaga anterior en la CPU
// * Est(n+1) = El estimado de la próxima ráfaga
func (p *Service) calcularSiguienteEstimacion(proceso *internal.Proceso) float64 {
	// Para procesos que nunca ejecutaron, usar directamente la estimación inicial
	if proceso.PCB.RafagaAnterior == nil {
		p.Log.Debug("Proceso nuevo - usando estimación inicial",
			log.IntAttr("pid", proceso.PCB.PID),
			log.AnyAttr("estimacion_inicial", float64(p.SjfConfig.InitialEstimate)),
		)
		return float64(p.SjfConfig.InitialEstimate * 1000) // Convertir a milisegundos
	}

	// Para procesos con historial, aplicar la fórmula SJF
	rafagaAnterior := float64(proceso.PCB.RafagaAnterior.Milliseconds())
	alpha := p.SjfConfig.Alpha
	estimacionAnterior := proceso.PCB.EstimacionAnterior

	nuevaEstimacion := alpha*rafagaAnterior + (1-alpha)*estimacionAnterior

	p.Log.Debug("Calculando nueva estimación SJF",
		log.IntAttr("pid", proceso.PCB.PID),
		log.AnyAttr("alpha", alpha),
		log.AnyAttr("rafaga_anterior", rafagaAnterior),
		log.AnyAttr("estimacion_anterior", estimacionAnterior),
		log.AnyAttr("nueva_estimacion", nuevaEstimacion),
	)

	return nuevaEstimacion
}

// evaluarDesalojo evalúa si el proceso nuevo debe desalojar algún proceso en ejecución
func (p *Service) evaluarDesalojo(procesoNuevo *internal.Proceso) *internal.Proceso {
	if len(p.Planificador.ExecQueue) == 0 {
		p.Log.Debug("No hay procesos en ExecQueue para evaluar desalojo")
		return nil
	}

	rafagaNueva := p.calcularSiguienteEstimacion(procesoNuevo)
	var (
		procesoADesalojar *internal.Proceso
		tiempoMax         float64 = -1 // Inicializar con un valor muy bajo
	)

	p.mutexExecQueue.Lock()
	for _, procesoEjecutando := range p.Planificador.ExecQueue {
		// Calcular tiempo restante del proceso en ejecución
		tiempoEjecutado := float64(time.Since(procesoEjecutando.PCB.MetricasTiempo[internal.EstadoExec].TiempoInicio).Milliseconds())
		tiempoAcumulado := float64(procesoEjecutando.PCB.MetricasTiempo[internal.EstadoExec].TiempoAcumulado.Milliseconds())
		rafagaEstimada := p.calcularSiguienteEstimacion(procesoEjecutando)
		tiempoRestante := rafagaEstimada - (tiempoAcumulado + tiempoEjecutado)

		//tiempoRestante := rafagaEstimada - tiempoEjecutado

		p.Log.Debug("Analizando proceso en ejecución",
			log.IntAttr("pid_ejecutando", procesoEjecutando.PCB.PID),
			log.AnyAttr("rafaga_estimada", rafagaEstimada),
			log.AnyAttr("tiempo_ejecutado", tiempoEjecutado),
			log.AnyAttr("tiempo_restante", tiempoRestante),
		)

		// Solo considerar procesos que aún tengan tiempo restante positivo
		// Un proceso que ya excedió su estimación no debería ser desalojado
		/*if tiempoRestante <= 0 {
			p.Log.Debug("Proceso ya excedió su estimación, no es candidato para desalojo",
				log.IntAttr("pid_ejecutando", procesoEjecutando.PCB.PID),
				log.AnyAttr("tiempo_restante", tiempoRestante),
			)
			continue
		}*/

		// Si el proceso nuevo tiene ráfaga menor que el tiempo restante
		if tiempoRestante > 0 && rafagaNueva < tiempoRestante && (tiempoRestante > tiempoMax) {
			procesoADesalojar = procesoEjecutando
			tiempoMax = tiempoRestante // Actualizar el máximo encontrado

			p.Log.Debug("Candidato a desalojo encontrado",
				log.IntAttr("pid_candidato", procesoEjecutando.PCB.PID),
			)
		}
	}

	if procesoADesalojar != nil {
		p.Log.Debug("DESALOJO SRT - Proceso seleccionado para desalojo",
			log.IntAttr("pid_desalojado", procesoADesalojar.PCB.PID),
			log.IntAttr("pid_nuevo", procesoNuevo.PCB.PID),
			log.AnyAttr("rafaga_nueva", rafagaNueva),
		)

		p.desalojarProceso(procesoADesalojar)
	} else {
		p.Log.Debug("No se encontró proceso para desalojar")
	}
	p.mutexExecQueue.Unlock()

	return procesoADesalojar
}

// actualizarRafagaAnterior actualiza la ráfaga anterior y estimación anterior del proceso
// Debe llamarse cada vez que un proceso deja de ejecutarse (por desalojo, IO, finalización, etc.)
func (p *Service) actualizarRafagaAnterior(proceso *internal.Proceso) {
	tiempoEjecutado := time.Since(proceso.PCB.MetricasTiempo[internal.EstadoExec].TiempoInicio)
	// Actualizar tiempo acumulado de ejecución
	if proceso.PCB.MetricasTiempo[internal.EstadoExec] != nil {
		proceso.PCB.MetricasTiempo[internal.EstadoExec].TiempoAcumulado += tiempoEjecutado
	}

	// Guardar la ráfaga real para el próximo cálculo
	if proceso.PCB.RafagaAnterior == nil {
		proceso.PCB.RafagaAnterior = &tiempoEjecutado
	} else {
		*proceso.PCB.RafagaAnterior = tiempoEjecutado
	}
	proceso.PCB.EstimacionAnterior = p.calcularSiguienteEstimacion(proceso)

	p.Log.Debug("Ráfaga anterior actualizada",
		log.IntAttr("pid", proceso.PCB.PID),
		log.AnyAttr("rafaga_ejecutada_ms", float64(tiempoEjecutado.Milliseconds())),
		log.AnyAttr("nueva_estimacion", proceso.PCB.EstimacionAnterior),
	)
}

// desalojarProceso desaloja un proceso de la CPU y lo devuelve a ReadyQueue
func (p *Service) desalojarProceso(proceso *internal.Proceso) {
	// Encontrar y liberar la CPU
	cpuFound := p.buscarCPUPorPID(proceso.PCB.PID)
	if cpuFound != nil {

		cpuFound.EnviarInterrupcion("Desalojo", false)
		// CRÍTICO: NO liberar CPU aquí - será liberada por DispatchProcess cuando termine
		p.Log.Debug("Interrupción de desalojo enviada",
			log.StringAttr("cpu_id", cpuFound.ID),
			log.IntAttr("pid", proceso.PCB.PID),
		)

		// Log obligatorio: Desalojo de SJF/SRT
		//"## (<PID>) - Desalojado por algoritmo SJF/SRT"
		p.Log.Info(fmt.Sprintf("## (%d) - Desalojado por algoritmo SJF/SRT", proceso.PCB.PID))

	} else {
		p.Log.Error("No se encontró CPU para el proceso a desalojar",
			log.IntAttr("pid", proceso.PCB.PID),
		)
		return
	}

	// Actualizar ráfaga anterior (incluye actualización de métricas de tiempo EXEC)
	//p.actualizarRafagaAnterior(proceso)

	// Remover de ExecQueue con protección de mutex
	//p.mutexExecQueue.Lock()
	var found bool
	p.Planificador.ExecQueue, found = p.removerDeCola(proceso.PCB.PID, p.Planificador.ExecQueue)
	if !found {
		p.Log.Error("🚨 Proceso no encontrado en ExecQueue durante desalojo",
			log.IntAttr("pid", proceso.PCB.PID),
		)
	}
	//p.mutexExecQueue.Unlock()

	// Devolver a ReadyQueue con protección de mutex
	p.mutexReadyQueue.Lock()
	p.Planificador.ReadyQueue = append(p.Planificador.ReadyQueue, proceso)

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

	// Notificar que hay un nuevo proceso en ReadyQueue tras desalojo
	select {
	case p.canalNuevoProcesoReady <- struct{}{}:
		p.Log.Debug("Notificación enviada al planificador tras desalojo",
			log.IntAttr("pid", proceso.PCB.PID),
		)
	default:
		// Canal lleno, no bloquear
		p.Log.Debug("Canal de notificación lleno, no se bloquea tras desalojo",
			log.IntAttr("pid", proceso.PCB.PID),
		)
	}
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
	// Validaciones iniciales
	if proceso == nil || proceso.PCB == nil || cpuAsignada == nil {
		p.Log.Error("CPU o proceso inválido al asignar a CPU",
			log.AnyAttr("proceso", proceso),
			log.AnyAttr("cpu", cpuAsignada),
		)
		return
	}

	// Verificar que el proceso no esté ya en ExecQueue
	p.mutexExecQueue.RLock()
	for _, proc := range p.Planificador.ExecQueue {
		if proc.PCB.PID == proceso.PCB.PID {
			p.Log.Error("Proceso ya está en ExecQueue, no se puede asignar nuevamente",
				log.IntAttr("pid", proceso.PCB.PID),
			)
			p.mutexExecQueue.RUnlock()
			return
		}
	}
	p.mutexExecQueue.RUnlock()

	// Mover proceso de READY a EXEC
	p.mutexReadyQueue.Lock()
	p.Planificador.ReadyQueue, _ = p.removerDeCola(proceso.PCB.PID, p.Planificador.ReadyQueue)
	timeReady := proceso.PCB.MetricasTiempo[internal.EstadoReady]
	if timeReady != nil {
		timeReady.TiempoAcumulado += time.Since(timeReady.TiempoInicio)
	}
	p.mutexReadyQueue.Unlock()

	p.mutexCPUsConectadas.Lock()
	cpuAsignada.Proceso.PID = proceso.PCB.PID
	cpuAsignada.Proceso.PC = proceso.PCB.PC
	cpuAsignada.Estado = false
	p.mutexCPUsConectadas.Unlock()

	// Agregar a ExecQueue
	p.mutexExecQueue.Lock()
	//Log obligatorio: Cambio de estado
	// "## (<PID>) Pasa del estado <ESTADO_ANTERIOR> al estado <ESTADO_ACTUAL>"
	p.Log.Info(fmt.Sprintf("## (%d) Pasa del estado READY al estado EXEC", proceso.PCB.PID))

	p.Planificador.ExecQueue = append(p.Planificador.ExecQueue, proceso)

	if proceso.PCB.MetricasTiempo[internal.EstadoExec] == nil {
		proceso.PCB.MetricasTiempo[internal.EstadoExec] = &internal.EstadoTiempo{}
	}
	proceso.PCB.MetricasTiempo[internal.EstadoExec].TiempoInicio = time.Now()
	proceso.PCB.MetricasEstado[internal.EstadoExec]++

	p.Log.Debug("Proceso asignado a CPU con SRT/SJF",
		log.IntAttr("PID", proceso.PCB.PID),
		log.StringAttr("CPU_ID", cpuAsignada.ID),
	)

	go func(cpuElegida *cpu.Cpu, procesoExec *internal.Proceso) {
		// Enviar proceso a la CPU
		newPC, motivo := cpuElegida.DispatchProcess()
		if procesoExec != nil && procesoExec.PCB != nil {
			procesoExec.PCB.PC = newPC
		}

		// Si hubo error al ejecutar el ciclo u otro problema, quitar de ExecQueue
		if motivo != "Proceso ejecutado exitosamente" {
			p.Log.Debug("Error al ejecutar proceso en CPU",
				log.IntAttr("PID", procesoExec.PCB.PID),
				log.StringAttr("motivo", motivo),
			)

			// Actualizar ráfaga anterior y estimación
			//p.actualizarRafagaAnterior(procesoExec)

			// Remover de ExecQueue solo si el proceso está allí
			/*p.mutexExecQueue.Lock()
			found := false
			p.Planificador.ExecQueue, found = p.removerDeCola(procesoExec.PCB.PID, p.Planificador.ExecQueue)
			if !found {
				p.Log.Error("🚨 Proceso no encontrado en ExecQueue durante asignarProcesoACPU",
					log.IntAttr("pid", procesoExec.PCB.PID),
				)
			}
			p.mutexExecQueue.Unlock()

			// Voler a agregar a ReadyQueue
			p.mutexReadyQueue.Lock()
			p.Planificador.ReadyQueue = append(p.Planificador.ReadyQueue, procesoExec)
			if procesoExec.PCB.MetricasTiempo[internal.EstadoReady] == nil {
				procesoExec.PCB.MetricasTiempo[internal.EstadoReady] = &internal.EstadoTiempo{}
			}
			procesoExec.PCB.MetricasTiempo[internal.EstadoReady].TiempoInicio = time.Now()
			procesoExec.PCB.MetricasEstado[internal.EstadoReady]++
			p.mutexReadyQueue.Unlock()*/

			// Log obligatorio: Cambio de estado
			// "## (<PID>) Pasa del estado <ESTADO_ANTERIOR> al estado <ESTADO_ACTUAL>"
			//p.Log.Info(fmt.Sprintf("## (%d) Pasa del estado EXEC al estado READY", procesoExec.PCB.PID))

			//p.canalNuevoProcesoReady <- struct{}{} // Notificar que hay un nuevo proceso en ReadyQueue

			//return
		}

		// Actualizar ráfaga anterior y estimación
		p.actualizarRafagaAnterior(procesoExec)

		// Liberar CPU usando semáforo
		p.LiberarCPU(cpuElegida)

	}(cpuAsignada, proceso)

	p.mutexExecQueue.Unlock()
}
