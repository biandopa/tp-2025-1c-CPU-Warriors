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

		// Drenar el canal para evitar notificaciones acumuladas innecesarias
	loopDrain:
		for {
			select {
			case <-p.canalNuevoProcesoReady:
				// Drenar notificación extra
			default:
				break loopDrain
			}
		}

		p.mutexSRT.Lock()
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
			if !p.evaluarDesalojo(procesoNuevo) && p.CantidadDeCpusDisponibles() > 0 {
				// Si no se realizó desalojo, asignar el proceso a una CPU libre
				if cpuLibre := p.BuscarCPUDisponible(); cpuLibre != nil {
					p.asignarProcesoACPU(procesoNuevo, cpuLibre)
				} else {
					p.Log.Debug("No hay CPUs libres para asignar el nuevo proceso (SJF)")
				}
			}
		}
		p.mutexSRT.Unlock()
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
		return float64(p.SjfConfig.InitialEstimate) // Ya está en milisegundos, no multiplicar
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
func (p *Service) evaluarDesalojo(procesoNuevo *internal.Proceso) bool {
	var desalojoRealizado bool

	// Si no hay procesos en ExecQueue, no hay nada que evaluar
	if len(p.Planificador.ExecQueue) == 0 {
		p.Log.Debug("No hay procesos en ExecQueue para evaluar desalojo")
		return desalojoRealizado
	}

	rafagaNueva := p.calcularSiguienteEstimacion(procesoNuevo)
	p.Log.Debug("🚀 Evaluando desalojo SRT",
		log.IntAttr("pid_nuevo", procesoNuevo.PCB.PID),
		log.AnyAttr("rafaga_nueva", rafagaNueva),
		log.IntAttr("procesos_en_exec", len(p.Planificador.ExecQueue)),
	)
	var (
		procesoADesalojar *internal.Proceso
		tiempoMax         float64 = -1 // Inicializar con un valor muy bajo
	)

	p.mutexExecQueue.Lock()
	for _, procesoEjecutando := range p.Planificador.ExecQueue {
		// Calcular tiempo restante del proceso en ejecución
		tiempoEjecutado := float64(time.Since(procesoEjecutando.PCB.MetricasTiempo[internal.EstadoExec].TiempoInicio).Milliseconds())
		//tiempoAcumulado := float64(procesoEjecutando.PCB.MetricasTiempo[internal.EstadoExec].TiempoAcumulado.Milliseconds())
		rafagaEstimada := p.calcularSiguienteEstimacion(procesoEjecutando)
		//tiempoRestante := rafagaEstimada - (tiempoAcumulado + tiempoEjecutado)

		tiempoRestante := rafagaEstimada - tiempoEjecutado

		p.Log.Debug("🔍 Analizando proceso en ejecución",
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

	p.mutexExecQueue.Unlock()

	if procesoADesalojar != nil {
		p.Log.Debug("🎯 DESALOJO SRT - Proceso seleccionado para desalojo",
			log.IntAttr("pid_desalojado", procesoADesalojar.PCB.PID),
			log.IntAttr("pid_nuevo", procesoNuevo.PCB.PID),
			log.AnyAttr("rafaga_nueva", rafagaNueva),
		)

		if cpuLiberada := p.desalojarProceso(procesoADesalojar); cpuLiberada != nil {
			p.asignarProcesoACPU(procesoNuevo, cpuLiberada)
			desalojoRealizado = true
		}
	} else {
		p.Log.Debug("❌ No se encontró proceso para desalojar")
	}

	return desalojoRealizado
}

// actualizarRafagaAnterior actualiza la ráfaga anterior y estimación anterior del proceso
// Debe llamarse cada vez que un proceso deja de ejecutarse (por desalojo, IO, finalización, etc.)
func (p *Service) actualizarRafagaAnterior(proceso *internal.Proceso) {
	tiempoEjecutado := time.Since(proceso.PCB.MetricasTiempo[internal.EstadoExec].TiempoInicio)

	// Actualizar tiempo acumulado de ejecución
	if proceso.PCB.MetricasTiempo[internal.EstadoExec] != nil {
		proceso.PCB.MetricasTiempo[internal.EstadoExec].TiempoAcumulado += tiempoEjecutado
	}

	// Calcular NUEVA estimación usando la ráfaga anterior actual
	nuevaEstimacion := p.calcularSiguienteEstimacion(proceso) // 1000

	// Después actualizar la ráfaga anterior con el tiempo recién ejecutado
	if proceso.PCB.RafagaAnterior == nil {
		proceso.PCB.RafagaAnterior = &tiempoEjecutado
	} else {
		*proceso.PCB.RafagaAnterior = tiempoEjecutado
	}

	// Guardar la nueva estimación calculada correctamente
	proceso.PCB.EstimacionAnterior = nuevaEstimacion

	p.Log.Debug("Ráfaga anterior actualizada",
		log.IntAttr("pid", proceso.PCB.PID),
		log.AnyAttr("rafaga_ejecutada_ms", float64(tiempoEjecutado.Milliseconds())),
		log.AnyAttr("nueva_estimacion", proceso.PCB.EstimacionAnterior),
	)
}

// desalojarProceso desaloja un proceso de la CPU y lo devuelve a ReadyQueue
func (p *Service) desalojarProceso(proceso *internal.Proceso) *cpu.Cpu {
	// Encontrar y liberar la CPU
	cpuFound := p.buscarCPUPorPID(proceso.PCB.PID)
	if cpuFound != nil {
		cpuFound.EnviarInterrupcion("Desalojo", false)
		// La CPU será liberada por DispatchProcess cuando termine, por lo que debemos esperar al semaforo
		<-p.CPUSemaphore

		// Verificar que el proceso AÚN esté en EXEC después de obtener el semáforo
		// (Esto previene condiciones de carrera con syscalls IO que pueden mover el proceso a BLOCKED)
		if procesoEnCola := p.BuscarProcesoEnCola(proceso.PCB.PID, "EXEC"); procesoEnCola == nil {
			p.Log.Info("El proceso ya no se encuentra en Exec, por lo que no hace falta desalojarlo",
				log.IntAttr("pid", proceso.PCB.PID),
			)
			return cpuFound // Retornar CPU liberada, pero no hacer el desalojo
		}

		//p.actualizarRafagaAnterior(proceso)

		// Remover de ExecQueue
		var found bool
		p.Planificador.ExecQueue, found = p.removerDeCola(proceso.PCB.PID, p.Planificador.ExecQueue)
		if found {
			proceso.PCB.MetricasTiempo[internal.EstadoExec].TiempoAcumulado +=
				time.Since(proceso.PCB.MetricasTiempo[internal.EstadoExec].TiempoInicio)
		}

		// Log obligatorio: Desalojo de SJF/SRT
		//"## (<PID>) - Desalojado por algoritmo SJF/SRT"
		p.Log.Info(fmt.Sprintf("## (%d) - Desalojado por algoritmo SJF/SRT", proceso.PCB.PID))

		// Devolver a ReadyQueue con protección de mutex
		p.mutexReadyQueue.Lock()
		p.Planificador.ReadyQueue = append(p.Planificador.ReadyQueue, proceso)

		// Actualizar métricas de Ready
		if proceso.PCB.MetricasTiempo[internal.EstadoReady] == nil {
			proceso.PCB.MetricasTiempo[internal.EstadoReady] = &internal.EstadoTiempo{}
		}
		proceso.PCB.MetricasTiempo[internal.EstadoReady].TiempoInicio = time.Now()
		proceso.PCB.MetricasEstado[internal.EstadoReady]++

		//Log obligatorio: Cambio de estado
		// "## (<PID>) Pasa del estado <ESTADO_ANTERIOR> al estado <ESTADO_ACTUAL>"
		p.Log.Info(fmt.Sprintf("## (%d) Pasa del estado EXEC al estado READY", proceso.PCB.PID))

		p.mutexReadyQueue.Unlock()

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

		return cpuFound

	}

	p.Log.Error("No se encontró CPU para el proceso a desalojar",
		log.IntAttr("pid", proceso.PCB.PID),
	)
	return nil
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
func (p *Service) asignarProcesoACPU(proceso *internal.Proceso, cpuAsignada *cpu.Cpu) bool {
	var asignado, removido bool

	// Validaciones iniciales
	if proceso == nil || proceso.PCB == nil || cpuAsignada == nil {
		p.Log.Error("CPU o proceso inválido al asignar a CPU",
			log.AnyAttr("proceso", proceso),
			log.AnyAttr("cpu", cpuAsignada),
		)
		return asignado
	}

	// Verificar que el proceso no esté ya en ExecQueue
	p.mutexExecQueue.RLock()
	for _, proc := range p.Planificador.ExecQueue {
		if proc != nil && proc.PCB.PID == proceso.PCB.PID {
			p.Log.Error("Proceso ya está en ExecQueue, no se puede asignar nuevamente",
				log.IntAttr("pid", proceso.PCB.PID),
			)
			//p.mutexExecQueue.RUnlock()
			return asignado
		}
	}
	p.mutexExecQueue.RUnlock()

	// Mover proceso de READY a EXEC
	p.mutexReadyQueue.Lock()
	p.Planificador.ReadyQueue, removido = p.removerDeCola(proceso.PCB.PID, p.Planificador.ReadyQueue)
	if !removido {
		p.Log.Error("🚨 Proceso no encontrado en ReadyQueue durante asignarProcesoACPU",
			log.IntAttr("pid", proceso.PCB.PID),
		)
		p.mutexReadyQueue.Unlock()
		return asignado
	}

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

	//Log obligatorio: Cambio de estado
	// "## (<PID>) Pasa del estado <ESTADO_ANTERIOR> al estado <ESTADO_ACTUAL>"
	p.Log.Info(fmt.Sprintf("## (%d) Pasa del estado READY al estado EXEC", proceso.PCB.PID))

	asignado = true

	go func(cpuElegida *cpu.Cpu, procesoExec *internal.Proceso) {
		// Enviar proceso a la CPU
		newPC, motivo := cpuElegida.DispatchProcess()
		if procesoExec != nil && procesoExec.PCB != nil {
			procesoExec.PCB.PC = newPC
		}

		// Si hubo error al ejecutar el ciclo u otro problema, quitar de ExecQueue
		if motivo != "Proceso ejecutado exitosamente" {
			p.Log.Info("Proceso desalojado",
				log.IntAttr("PID", procesoExec.PCB.PID),
				log.StringAttr("motivo", motivo),
			)
		}

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
		//}

		// Actualizar ráfaga anterior y estimación
		p.actualizarRafagaAnterior(procesoExec)

		// Liberar CPU usando semáforo
		p.LiberarCPU(cpuElegida)
	}(cpuAsignada, proceso)

	p.mutexExecQueue.Unlock()

	return asignado
}
