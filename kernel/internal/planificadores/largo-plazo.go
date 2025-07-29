package planificadores

import (
	"bufio"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/sisoputnfrba/tp-golang/kernel/internal"
	"github.com/sisoputnfrba/tp-golang/utils/log"
)

const (
	PlanificadorEstadoStop  = "STOP"
	PlanificadorEstadoStart = "START"
)

// PlanificadorLargoPlazo realiza las funciones correspondientes al planificador de largo plazo.
func (p *Service) PlanificadorLargoPlazo() {
	estado := PlanificadorEstadoStop

	// Lanzamos una goroutine que espera el Enter
	go func() {
		reader := bufio.NewReader(os.Stdin)
		_, _ = reader.ReadString('\n') // Espera hasta que se presione Enter
		p.CanalEnter <- struct{}{}     // Envía una señal al canal
		estado = PlanificadorEstadoStart
	}()

	// Se queda escuchando hasta que el usuario presione la tecla ENTER por consola para iniciar el planificador
	<-p.CanalEnter

	if estado == PlanificadorEstadoStart {
		for {

			procesoNew := <-p.CanalNuevoProcesoNew

			//agrego al planificador
			switch p.LargoPlazoAlgorithm {
			case "FIFO":
				p.PlanificadorLargoPlazoFIFO(procesoNew)
			case "PMCP":
				p.PlanificadorLargoPlazoPMCP(procesoNew)
			default:
				p.Log.Warn("Algoritmo de largo plazo no reconocido")
			}

			p.CheckearEspacioEnMemoria()
		}
	}
}

func (p *Service) PlanificadorLargoPlazoFIFO(proceso *internal.Proceso) {

	p.mutexNewQueue.Lock()
	p.Planificador.NewQueue = append([]*internal.Proceso{proceso}, p.Planificador.NewQueue...)
	p.mutexNewQueue.Unlock()

}

func (p *Service) PlanificadorLargoPlazoPMCP(proceso *internal.Proceso) {
	sizeProcesoEntrante, _ := strconv.Atoi(proceso.PCB.Tamanio)

	p.mutexNewQueue.Lock()
	var yaLoAgregue = true

	for i, procesoEncolado := range p.Planificador.NewQueue {

		sizeProcesoEncolado, _ := strconv.Atoi(procesoEncolado.PCB.Tamanio)
		if sizeProcesoEntrante < sizeProcesoEncolado {

			// Si el proceso entrante es más chico que el encolado, lo agrego antes
			p.Planificador.NewQueue = append(
				p.Planificador.NewQueue[:i],
				append([]*internal.Proceso{proceso}, p.Planificador.NewQueue[i:]...)...,
			)

			yaLoAgregue = false
			break
		}
	}

	if yaLoAgregue {
		p.Planificador.NewQueue = append([]*internal.Proceso{proceso}, p.Planificador.NewQueue...)
	}

	p.mutexNewQueue.Unlock()

}

func (p *Service) CheckearEspacioEnMemoria() {
	// Priorizamos los procesos suspendidos ready
	// Usar scope limitado de mutex para evitar deadlocks
	var procesosAMover []*internal.Proceso

	p.mutexSuspReadyQueue.Lock()
	for _, proceso := range p.Planificador.SuspReadyQueue {
		if p.Memoria.ConsultarEspacio(proceso.PCB.Tamanio, proceso.PCB.PID) {
			procesosAMover = append(procesosAMover, proceso)
		} else {
			// Si no hay espacio, salir del loop
			break
		}
	}
	p.mutexSuspReadyQueue.Unlock()

	// Procesar movimientos fuera del lock para evitar deadlocks
	for _, proceso := range procesosAMover {
		// Remover de SuspReadyQueue
		p.mutexSuspReadyQueue.Lock()
		var removido bool
		p.Planificador.SuspReadyQueue, removido = p.removerDeCola(proceso.PCB.PID, p.Planificador.SuspReadyQueue)
		p.mutexSuspReadyQueue.Unlock()

		if !removido {
			p.Log.Error("🚨 Proceso no encontrado en SuspReadyQueue durante carga a memoria",
				log.IntAttr("pid", proceso.PCB.PID),
			)
			continue
		}

		// Actualizar métricas de tiempo
		if proceso.PCB.MetricasTiempo[internal.EstadoSuspReady] == nil {
			proceso.PCB.MetricasTiempo[internal.EstadoSuspReady] = &internal.EstadoTiempo{}
		}
		timeSusp := proceso.PCB.MetricasTiempo[internal.EstadoSuspReady]
		timeSusp.TiempoAcumulado = timeSusp.TiempoAcumulado + time.Since(timeSusp.TiempoInicio)

		// Agregar a ReadyQueue
		p.mutexReadyQueue.Lock()
		p.Planificador.ReadyQueue = append(p.Planificador.ReadyQueue, proceso)
		if proceso.PCB.MetricasTiempo[internal.EstadoReady] == nil {
			proceso.PCB.MetricasTiempo[internal.EstadoReady] = &internal.EstadoTiempo{}
		}
		proceso.PCB.MetricasTiempo[internal.EstadoReady].TiempoInicio = time.Now()
		proceso.PCB.MetricasEstado[internal.EstadoReady]++
		p.mutexReadyQueue.Unlock()

		//Log obligatorio: Cambio de estado
		p.Log.Info(fmt.Sprintf("## (%d) Pasa del estado SUSP.READY al estado READY", proceso.PCB.PID))

		// Enviar señal al canal de corto plazo para procesos suspendidos
		p.Log.Debug("Enviando señal al canal de corto plazo (SUSP.READY -> READY)",
			log.IntAttr("pid", proceso.PCB.PID))

		// Notificar al planificador de corto plazo
		p.canalNuevoProcesoReady <- struct{}{}
	}

	// Si no hay procesos suspendidos, revisar NewQueue
	p.mutexSuspReadyQueue.RLock()
	suspReadyVacia := len(p.Planificador.SuspReadyQueue) == 0
	p.mutexSuspReadyQueue.RUnlock()

	if suspReadyVacia {
		// Usar scope limitado de mutex para evitar deadlocks
		var procesosACargar []*internal.Proceso

		p.mutexNewQueue.Lock()
		for _, proceso := range p.Planificador.NewQueue {
			if p.Memoria.ConsultarEspacio(proceso.PCB.Tamanio, proceso.PCB.PID) {
				procesosACargar = append(procesosACargar, proceso)
			} else {
				p.Log.Debug("No hay espacio en memoria para el proceso",
					log.IntAttr("pid", proceso.PCB.PID))
				break // Si no hay espacio para este proceso, no habrá para los siguientes
			}
		}
		p.mutexNewQueue.Unlock()

		// Procesar carga fuera del lock para evitar deadlocks
		for _, proceso := range procesosACargar {
			// Remover de NewQueue
			p.mutexNewQueue.Lock()
			var removido bool
			p.Planificador.NewQueue, removido = p.removerDeCola(proceso.PCB.PID, p.Planificador.NewQueue)
			p.mutexNewQueue.Unlock()

			if !removido {
				p.Log.Error("🚨 Proceso no encontrado en NewQueue durante carga a memoria",
					log.IntAttr("pid", proceso.PCB.PID),
				)
				continue
			}

			// Actualizar métricas
			if proceso.PCB.MetricasTiempo[internal.EstadoNew] == nil {
				proceso.PCB.MetricasTiempo[internal.EstadoNew] = &internal.EstadoTiempo{}
			}
			timeNew := proceso.PCB.MetricasTiempo[internal.EstadoNew]
			timeNew.TiempoAcumulado = timeNew.TiempoAcumulado + time.Since(timeNew.TiempoInicio)

			if proceso.PCB.MetricasTiempo[internal.EstadoReady] == nil {
				proceso.PCB.MetricasTiempo[internal.EstadoReady] = &internal.EstadoTiempo{}
			}

			// Cargar en memoria
			p.Memoria.CargarProcesoEnMemoriaDeSistema(proceso.PCB.NombreArchivo, proceso.PCB.PID)

			// Agregar a ReadyQueue
			p.mutexReadyQueue.Lock()
			p.Planificador.ReadyQueue = append(p.Planificador.ReadyQueue, proceso)
			proceso.PCB.MetricasTiempo[internal.EstadoReady].TiempoInicio = time.Now()
			proceso.PCB.MetricasEstado[internal.EstadoReady]++
			p.mutexReadyQueue.Unlock()

			//Log obligatorio: Cambio de estado
			p.Log.Info(fmt.Sprintf("## (%d) Pasa del estado NEW al estado READY", proceso.PCB.PID))

			p.Log.Debug("Enviando señal al canal de corto plazo",
				log.IntAttr("pid", proceso.PCB.PID))

			// Notificar al planificador de corto plazo
			p.canalNuevoProcesoReady <- struct{}{}
		}
	}
}

func (p *Service) FinalizarProceso(pid int) {
	// 1. Buscar el proceso en la cola de exec (verificación inicial sin lock)
	var proceso *internal.Proceso

	// Verificación inicial para ver si el proceso existe
	procesoEncontrado := false
	for _, proc := range p.Planificador.ExecQueue {
		if proc.PCB.PID == pid {
			proceso = proc
			procesoEncontrado = true
			break
		}
	}

	if !procesoEncontrado {
		p.Log.Debug("No se encontró el proceso en la cola de exec",
			log.IntAttr("PID", pid),
		)
		return
	}

	// 2. Notificar a Memoria
	status, err := p.Memoria.FinalizarProceso(proceso.PCB.PID)
	if err != nil || status != http.StatusOK {
		p.Log.Error("Error al finalizar proceso en memoria",
			log.ErrAttr(err),
			log.IntAttr("PID", proceso.PCB.PID),
		)
		return
	}

	// 3. PRIMERO: Remover de ExecQueue antes de actualizar ráfaga (evita deadlock)
	p.mutexExecQueue.Lock()
	var removido bool
	p.Planificador.ExecQueue, removido = p.removerDeCola(pid, p.Planificador.ExecQueue)
	if !removido {
		p.Log.Error("🚨 Proceso a finalizar no estaba en ExecQueue",
			log.IntAttr("pid", pid),
		)
	}
	p.mutexExecQueue.Unlock()

	// 4. DESPUÉS: Actualizar ráfaga anterior (IMPORTANTE para SRT)
	//p.actualizarRafagaAnterior(proceso)

	// 5. Cambiar el estado de la CPU y notificar al planificador
	cpuFound := p.buscarCPUPorPID(proceso.PCB.PID)
	if cpuFound != nil {
		p.LiberarCPU(cpuFound) // Usar función centralizada que incluye notificación al planificador
	}

	if proceso.PCB.MetricasTiempo[internal.EstadoExit] == nil {
		proceso.PCB.MetricasTiempo[internal.EstadoExit] = &internal.EstadoTiempo{}
	}
	proceso.PCB.MetricasTiempo[internal.EstadoExit].TiempoInicio = time.Now()

	// 6. Cambiar el estado del proceso a EXIT (las métricas de EXEC ya están actualizadas por actualizarRafagaAnterior)
	proceso.PCB.MetricasTiempo[internal.EstadoExit].TiempoAcumulado = time.Since(proceso.PCB.MetricasTiempo[internal.EstadoExit].TiempoInicio)
	proceso.PCB.MetricasEstado[internal.EstadoExec]++
	proceso.PCB.MetricasEstado[internal.EstadoExit]++

	// 7. Loguear métricas
	//Log obligatorio: Cambio de estado
	// "## (<PID>) Pasa del estado <ESTADO_ANTERIOR> al estado <ESTADO_ACTUAL>"
	p.Log.Info(fmt.Sprintf("## (%d) Pasa del estado EXEC al estado EXIT", proceso.PCB.PID))

	//Log obligatorio: Finalización de proceso
	//"## (<PID>) - Finaliza el proceso"
	p.Log.Info(fmt.Sprintf("## (%d) Finaliza el proceso", proceso.PCB.PID))

	// Log obligatorio: Métricas de Estado
	//"## (<PID>) - Métricas de estado: NEW (NEW_COUNT) (NEW_TIME), READY (READY_COUNT) (READY_TIME), …"
	p.Log.Info(fmt.Sprintf("## (%d) - Métricas de estado: NEW %d %d, READY %d %d, "+
		"EXEC %d %d, BLOCKED %d %d, SUSP. BLOCKED %d %d, SUSP. READY %d %d, EXIT %d %d",
		proceso.PCB.PID,
		proceso.PCB.MetricasEstado[internal.EstadoNew],
		proceso.PCB.MetricasTiempo[internal.EstadoNew].TiempoAcumulado.Milliseconds(),
		proceso.PCB.MetricasEstado[internal.EstadoReady],
		proceso.PCB.MetricasTiempo[internal.EstadoReady].TiempoAcumulado.Milliseconds(),
		proceso.PCB.MetricasEstado[internal.EstadoExec],
		proceso.PCB.MetricasTiempo[internal.EstadoExec].TiempoAcumulado.Milliseconds(),
		proceso.PCB.MetricasEstado[internal.EstadoBloqueado],
		proceso.PCB.MetricasTiempo[internal.EstadoBloqueado].TiempoAcumulado.Milliseconds(),
		proceso.PCB.MetricasEstado[internal.EstadoSuspBloqueado],
		proceso.PCB.MetricasTiempo[internal.EstadoSuspBloqueado].TiempoAcumulado.Milliseconds(),
		proceso.PCB.MetricasEstado[internal.EstadoSuspReady],
		proceso.PCB.MetricasTiempo[internal.EstadoSuspReady].TiempoAcumulado.Milliseconds(),
		proceso.PCB.MetricasEstado[internal.EstadoExit],
		proceso.PCB.MetricasTiempo[internal.EstadoExit].TiempoAcumulado.Milliseconds(),
	),
	)

	// 8. Checkear si hay procesos suspendidos que puedan volver a memoria
	p.CheckearEspacioEnMemoria()
}

// FinalizarProcesoEnCualquierCola busca un proceso en todas las colas y lo finaliza
// Esta función es útil cuando el proceso puede estar en BLOCKED, SUSP.BLOCKED, etc.
func (p *Service) FinalizarProcesoEnCualquierCola(pid int) {
	var (
		proceso        *internal.Proceso
		estadoAnterior string
	)

	// 1. Buscar el proceso en todas las colas posibles
	// Primero en EXEC (comportamiento normal) - PROTEGER CON MUTEX
	p.mutexExecQueue.Lock()
	for _, proc := range p.Planificador.ExecQueue {
		if proc.PCB.PID == pid {
			proceso = proc
			estadoAnterior = "EXEC"

			// Sacarlo de EXEC PRIMERO usando función segura
			var removido bool
			p.Planificador.ExecQueue, removido = p.removerDeCola(pid, p.Planificador.ExecQueue)
			if !removido {
				p.Log.Error("🚨 CRÍTICO: Proceso no encontrado al finalizar en cualquier cola",
					log.IntAttr("pid", pid),
				)
			}
			break
		}
	}
	p.mutexExecQueue.Unlock()

	// Si encontré proceso en EXEC, actualizar y liberar CPU
	if proceso != nil {
		// DESPUÉS actualizar ráfaga anterior (IMPORTANTE para SRT - pero sin deadlock)
		//p.actualizarRafagaAnterior(proceso)

		// Liberar CPU
		cpuFound := p.buscarCPUPorPID(proceso.PCB.PID)
		if cpuFound != nil {
			p.LiberarCPU(cpuFound) // Usar función centralizada que incluye notificación al planificador
		}
	}

	// Si no está en EXEC, buscar en BLOCKED
	if proceso == nil {
		p.mutexBlockQueue.Lock()
		for _, proc := range p.Planificador.BlockQueue {
			if proc.PCB.PID == pid {
				proceso = proc
				estadoAnterior = "BLOCKED"
				break
			}
		}

		// Sacarlo de BLOCKED
		if proceso != nil {
			var removido bool
			p.Planificador.BlockQueue, removido = p.removerDeCola(pid, p.Planificador.BlockQueue)
			if !removido {
				p.Log.Error("🚨 Proceso no encontrado en BlockQueue durante finalización",
					log.IntAttr("pid", pid),
				)
			}
		}
		p.mutexBlockQueue.Unlock()
	}

	// Si no está en BLOCKED, buscar en SUSP.BLOCKED
	if proceso == nil {
		p.mutexSuspBlockQueue.Lock()
		for _, proc := range p.Planificador.SuspBlockQueue {
			if proc.PCB.PID == pid {
				proceso = proc
				estadoAnterior = "SUSP.BLOCKED"
				break
			}
		}

		// Sacarlo de SUSP.BLOCKED
		if proceso != nil {
			var removido bool
			p.Planificador.SuspBlockQueue, removido = p.removerDeCola(pid, p.Planificador.SuspBlockQueue)
			if !removido {
				p.Log.Error("🚨 Proceso no encontrado en SuspBlockQueue durante finalización",
					log.IntAttr("pid", pid),
				)
			}
		}
		p.mutexSuspBlockQueue.Unlock()
	}

	// Si no está en ninguna de las anteriores, buscar en READY
	if proceso == nil {
		p.mutexReadyQueue.Lock()
		for _, proc := range p.Planificador.ReadyQueue {
			if proc.PCB.PID == pid {
				proceso = proc
				estadoAnterior = "READY"
				break
			}
		}

		// Sacarlo de READY usando función segura
		if proceso != nil {
			var removido bool
			p.Planificador.ReadyQueue, removido = p.removerDeCola(pid, p.Planificador.ReadyQueue)
			if !removido {
				p.Log.Debug("🚨 Proceso no encontrado en ReadyQueue durante finalización",
					log.IntAttr("pid", pid),
				)
			}
		}
		p.mutexReadyQueue.Unlock()
	}

	// Si no está en READY, buscar en SUSP.READY
	if proceso == nil {
		p.mutexSuspReadyQueue.Lock()
		for _, proc := range p.Planificador.SuspReadyQueue {
			if proc.PCB.PID == pid {
				proceso = proc
				estadoAnterior = "SUSP.READY"
				break
			}
		}

		// Sacarlo de SUSP.READ
		if proceso != nil {
			var removido bool
			p.Planificador.SuspReadyQueue, removido = p.removerDeCola(pid, p.Planificador.SuspReadyQueue)
			if !removido {
				p.Log.Error("🚨 Proceso no encontrado en SuspReadyQueue durante finalización",
					log.IntAttr("pid", pid),
				)
			}
		}
		p.mutexSuspReadyQueue.Unlock()
	}

	if proceso == nil {
		p.Log.Debug("No se encontró el proceso en ninguna cola",
			log.IntAttr("PID", pid),
		)
		return
	}

	// 2. Notificar a Memoria
	status, err := p.Memoria.FinalizarProceso(proceso.PCB.PID)
	if err != nil || status != http.StatusOK {
		p.Log.Debug("Error al finalizar proceso en memoria",
			log.ErrAttr(err),
			log.IntAttr("PID", proceso.PCB.PID),
		)
		return
	}

	// 3. Actualizar métricas según el estado anterior
	proceso.PCB.MetricasEstado[internal.EstadoExit]++
	if proceso.PCB.MetricasTiempo[internal.EstadoExit] == nil {
		proceso.PCB.MetricasTiempo[internal.EstadoExit] = &internal.EstadoTiempo{}
	}
	proceso.PCB.MetricasTiempo[internal.EstadoExit].TiempoInicio = time.Now()

	// Actualizar tiempo acumulado del estado anterior
	switch estadoAnterior {
	case "EXEC":
		// Ya se actualizó en actualizarRafagaAnterior() - no hacer nada adicional
	case "BLOCKED":
		if proceso.PCB.MetricasTiempo[internal.EstadoBloqueado] != nil {
			proceso.PCB.MetricasTiempo[internal.EstadoBloqueado].TiempoAcumulado += time.Since(proceso.PCB.MetricasTiempo[internal.EstadoBloqueado].TiempoInicio)
		}
	case "SUSP.BLOCKED":
		if proceso.PCB.MetricasTiempo[internal.EstadoSuspBloqueado] != nil {
			proceso.PCB.MetricasTiempo[internal.EstadoSuspBloqueado].TiempoAcumulado += time.Since(proceso.PCB.MetricasTiempo[internal.EstadoSuspBloqueado].TiempoInicio)
		}
	case "READY":
		if proceso.PCB.MetricasTiempo[internal.EstadoReady] != nil {
			proceso.PCB.MetricasTiempo[internal.EstadoReady].TiempoAcumulado += time.Since(proceso.PCB.MetricasTiempo[internal.EstadoReady].TiempoInicio)
		}
	case "SUSP.READY":
		if proceso.PCB.MetricasTiempo[internal.EstadoSuspReady] != nil {
			proceso.PCB.MetricasTiempo[internal.EstadoSuspReady].TiempoAcumulado += time.Since(proceso.PCB.MetricasTiempo[internal.EstadoSuspReady].TiempoInicio)
		}
	}

	//Log obligatorio: Cambio de estado
	p.Log.Info(fmt.Sprintf("## (%d) Pasa del estado %s al estado EXIT", proceso.PCB.PID, estadoAnterior))

	//Log obligatorio: Finalización de proceso
	p.Log.Info(fmt.Sprintf("## (%d) Finaliza el proceso", proceso.PCB.PID))

	// Log obligatorio: Métricas de Estado
	p.Log.Info(fmt.Sprintf("## (%d) - Métricas de estado: NEW %d %d, READY %d %d, "+
		"EXEC %d %d, BLOCKED %d %d, SUSP. BLOCKED %d %d, SUSP. READY %d %d, EXIT %d %d",
		proceso.PCB.PID,
		proceso.PCB.MetricasEstado[internal.EstadoNew],
		proceso.PCB.MetricasTiempo[internal.EstadoNew].TiempoAcumulado.Milliseconds(),
		proceso.PCB.MetricasEstado[internal.EstadoReady],
		proceso.PCB.MetricasTiempo[internal.EstadoReady].TiempoAcumulado.Milliseconds(),
		proceso.PCB.MetricasEstado[internal.EstadoExec],
		proceso.PCB.MetricasTiempo[internal.EstadoExec].TiempoAcumulado.Milliseconds(),
		proceso.PCB.MetricasEstado[internal.EstadoBloqueado],
		proceso.PCB.MetricasTiempo[internal.EstadoBloqueado].TiempoAcumulado.Milliseconds(),
		proceso.PCB.MetricasEstado[internal.EstadoSuspBloqueado],
		proceso.PCB.MetricasTiempo[internal.EstadoSuspBloqueado].TiempoAcumulado.Milliseconds(),
		proceso.PCB.MetricasEstado[internal.EstadoSuspReady],
		proceso.PCB.MetricasTiempo[internal.EstadoSuspReady].TiempoAcumulado.Milliseconds(),
		proceso.PCB.MetricasEstado[internal.EstadoExit],
		proceso.PCB.MetricasTiempo[internal.EstadoExit].TiempoAcumulado.Milliseconds(),
	),
	)

	// 4. Checkear si hay procesos suspendidos que puedan volver a memoria
	p.CheckearEspacioEnMemoria()
}
