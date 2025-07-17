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
	p.Memoria.CargarProcesoEnMemoriaDeSistema(proceso.PCB.NombreArchivo, proceso.PCB.PID)
	p.mutexNewQueue.Unlock()

}

func (p *Service) PlanificadorLargoPlazoPMCP(proceso *internal.Proceso) {

	sizeProcesoEntrante, _ := strconv.Atoi(proceso.PCB.Tamanio)

	p.mutexNewQueue.Lock()
	var yaLoAgregue = true

	for i, procesoEncolado := range p.Planificador.NewQueue {

		sizeProcesoEncolado, _ := strconv.Atoi(procesoEncolado.PCB.Tamanio)
		if sizeProcesoEntrante < sizeProcesoEncolado {

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
		p.Memoria.CargarProcesoEnMemoriaDeSistema(proceso.PCB.NombreArchivo, proceso.PCB.PID)
	}

	p.mutexNewQueue.Unlock()

}

func (p *Service) CheckearEspacioEnMemoria() {
	// Priorizamos los procesos suspendidos ready
	p.mutexSuspReadyQueue.Lock()
	i := 0
	for i < len(p.Planificador.SuspReadyQueue) {
		proceso := p.Planificador.SuspReadyQueue[i]
		if p.Memoria.ConsultarEspacio(proceso.PCB.Tamanio, proceso.PCB.PID) {
			// Si el proceso se carga en memoria, lo muevo a la cola de ready
			// y lo elimino de la cola de suspendidos ready

			// Remover el proceso de la cola usando índice
			p.Planificador.SuspReadyQueue = append(p.Planificador.SuspReadyQueue[:i], p.Planificador.SuspReadyQueue[i+1:]...)

			if proceso.PCB.MetricasTiempo[internal.EstadoSuspReady] == nil {
				proceso.PCB.MetricasTiempo[internal.EstadoSuspReady] = &internal.EstadoTiempo{}
			}
			timeSusp := proceso.PCB.MetricasTiempo[internal.EstadoSuspReady]
			timeSusp.TiempoAcumulado = timeSusp.TiempoAcumulado + time.Since(timeSusp.TiempoInicio)

			// Agrego el proceso a la cola de ready
			p.mutexReadyQueue.Lock()
			p.Planificador.ReadyQueue = append(p.Planificador.ReadyQueue, proceso)
			if proceso.PCB.MetricasTiempo[internal.EstadoReady] == nil {
				proceso.PCB.MetricasTiempo[internal.EstadoReady] = &internal.EstadoTiempo{}
			}
			proceso.PCB.MetricasTiempo[internal.EstadoReady].TiempoInicio = time.Now()

			proceso.PCB.MetricasEstado[internal.EstadoReady]++
			p.mutexReadyQueue.Unlock()

			// "## (<PID>) Pasa del estado <ESTADO_ANTERIOR> al estado <ESTADO_ACTUAL>"
			p.Log.Info(fmt.Sprintf("## (%d) Pasa del estado SUSP.READY al estado READY", proceso.PCB.PID))

			// Enviar señal al canal de corto plazo para procesos suspendidos
			p.Log.Debug("Enviando señal al canal de corto plazo (SUSP.READY -> READY)",
				log.IntAttr("pid", proceso.PCB.PID))

			p.canalNuevoProcesoReady <- struct{}{}

			// No incrementar i porque removimos un elemento
		} else {
			// Si no hay espacio, salir del loop
			break
		}
	}
	p.mutexSuspReadyQueue.Unlock()

	if len(p.Planificador.SuspReadyQueue) == 0 {
		p.mutexNewQueue.Lock()
		i := 0
		for i < len(p.Planificador.NewQueue) {
			proceso := p.Planificador.NewQueue[i]
			if p.Memoria.ConsultarEspacio(proceso.PCB.Tamanio, proceso.PCB.PID) {
				// Si el proceso se carga en memoria, lo muevo a la cola de ready
				// y lo elimino de la cola de new

				// Remover el proceso de la cola usando índice
				p.Planificador.NewQueue = append(p.Planificador.NewQueue[:i], p.Planificador.NewQueue[i+1:]...)

				if proceso.PCB.MetricasTiempo[internal.EstadoNew] == nil {
					proceso.PCB.MetricasTiempo[internal.EstadoNew] = &internal.EstadoTiempo{}
				}
				timeNew := proceso.PCB.MetricasTiempo[internal.EstadoNew]
				timeNew.TiempoAcumulado = timeNew.TiempoAcumulado + time.Since(timeNew.TiempoInicio)

				if proceso.PCB.MetricasTiempo[internal.EstadoReady] == nil {
					proceso.PCB.MetricasTiempo[internal.EstadoReady] = &internal.EstadoTiempo{}
				}

				// Primero agrego el proceso a la cola de ready
				p.mutexReadyQueue.Lock()
				p.Planificador.ReadyQueue = append(p.Planificador.ReadyQueue, proceso)
				proceso.PCB.MetricasTiempo[internal.EstadoReady].TiempoInicio = time.Now()
				proceso.PCB.MetricasEstado[internal.EstadoReady]++
				p.mutexReadyQueue.Unlock()

				// "## (<PID>) Pasa del estado <ESTADO_ANTERIOR> al estado <ESTADO_ACTUAL>"
				p.Log.Info(fmt.Sprintf("## (%d) Pasa del estado NEW al estado READY", proceso.PCB.PID))

				// Luego, envío la señal para que el planificador de corto plazo pueda ejecutar el proceso
				p.Log.Debug("Enviando señal al canal de corto plazo",
					log.IntAttr("pid", proceso.PCB.PID))

				p.canalNuevoProcesoReady <- struct{}{}

				// No incrementar i porque removimos un elemento
			} else {
				p.Log.Debug("No hay espacio en memoria para el proceso",
					log.IntAttr("pid", proceso.PCB.PID))
				break
			}
		}
		p.mutexNewQueue.Unlock()
	}
}

func (p *Service) FinalizarProceso(pid int) {
	// 1. Buscar el proceso en la cola de exec
	var (
		proceso       *internal.Proceso
		lugarColaExec int
	)

	for i, proc := range p.Planificador.ExecQueue {
		if proc.PCB.PID == pid {
			proceso = proc
			lugarColaExec = i
			break
		}
	}

	if proceso == nil {
		p.Log.Error("No se encontró el proceso en la cola de exec",
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

	// 3. Lo saco de la cola de exec
	p.Planificador.ExecQueue = append(p.Planificador.ExecQueue[:lugarColaExec], p.Planificador.ExecQueue[lugarColaExec+1:]...)

	// 4. Cambiar el estado de la CPU
	cpuFound := p.buscarCPUPorPID(proceso.PCB.PID)
	if cpuFound != nil {
		cpuFound.Estado = true
		// Informo al channel de que la CPU esta libre
		p.CPUSemaphore <- struct{}{}
	}

	// 5. Cambiar el estado del proceso a EXIT
	proceso.PCB.MetricasEstado[internal.EstadoExit]++
	if proceso.PCB.MetricasTiempo[internal.EstadoExit] == nil {
		proceso.PCB.MetricasTiempo[internal.EstadoExit] = &internal.EstadoTiempo{}
	}
	proceso.PCB.MetricasTiempo[internal.EstadoExit].TiempoInicio = time.Now()
	proceso.PCB.MetricasTiempo[internal.EstadoExit].TiempoAcumulado = time.Since(proceso.PCB.MetricasTiempo[internal.EstadoExec].TiempoInicio)
	proceso.PCB.MetricasTiempo[internal.EstadoExec].TiempoAcumulado += time.Since(proceso.PCB.MetricasTiempo[internal.EstadoExec].TiempoInicio)
	proceso.PCB.MetricasEstado[internal.EstadoExec]++
	proceso.PCB.MetricasEstado[internal.EstadoExit]++

	// 6. Loguear métricas (acá deberías tenerlas guardadas en el PCB)

	//Log obligatorio: Cambio de estado
	// "## (<PID>) Pasa del estado <ESTADO_ANTERIOR> al estado <ESTADO_ACTUAL>"
	p.Log.Info(fmt.Sprintf("## (%d) Pasa del estado EXEC al estado EXIT", proceso.PCB.PID))

	//Log obligatorio: Finalización de proceso
	//"## (<PID>) - Finaliza el proceso"
	p.Log.Info(fmt.Sprintf("## (%d) Finaliza el proceso", proceso.PCB.PID))

	// Log obligatorio: Métricas de Estado
	//"## (<PID>) - Métricas de estado: NEW (NEW_COUNT) (NEW_TIME), READY (READY_COUNT) (READY_TIME), …"
	p.Log.Info(fmt.Sprintf("## (%d) - Métricas de estado", proceso.PCB.PID),
		log.AnyAttr("metricas_estado", proceso.PCB.MetricasEstado),
		log.AnyAttr("metricas_tiempo", proceso.PCB.MetricasTiempo),
	)

	// 7. Liberar PCB
	proceso.PCB = nil // Libero el PCB asociado al proceso

	// 8. Checkear si hay procesos suspendidos que puedan volver a memoria
	p.CheckearEspacioEnMemoria()

	/*// 9. Le avisamos al channel de que puede ejecutar el algoritmo de largo plazo
	p.canalNuevoProcesoReady <- struct{}{}
	p.canalRafagaActualizada <- struct{}{}*/
}

// FinalizarProcesoEnCualquierCola busca un proceso en todas las colas y lo finaliza
// Esta función es útil cuando el proceso puede estar en BLOCKED, SUSP.BLOCKED, etc.
func (p *Service) FinalizarProcesoEnCualquierCola(pid int) {
	var proceso *internal.Proceso
	var estadoAnterior string

	p.Log.Info("Procesos en las colas antes de finalizar",
		log.AnyAttr("ExecQueue", p.Planificador.ExecQueue),
		log.AnyAttr("BlockQueue", p.Planificador.BlockQueue),
		log.AnyAttr("SuspBlockQueue", p.Planificador.SuspBlockQueue),
		log.AnyAttr("ReadyQueue", p.Planificador.ReadyQueue),
		log.AnyAttr("SuspReadyQueue", p.Planificador.SuspReadyQueue),
	)
	// 1. Buscar el proceso en todas las colas posibles
	// Primero en EXEC (comportamiento normal)
	for i, proc := range p.Planificador.ExecQueue {
		if proc.PCB.PID == pid {
			proceso = proc
			estadoAnterior = "EXEC"
			// Sacarlo de EXEC
			p.Planificador.ExecQueue = append(p.Planificador.ExecQueue[:i], p.Planificador.ExecQueue[i+1:]...)
			// Liberar CPU
			cpuFound := p.buscarCPUPorPID(proceso.PCB.PID)
			if cpuFound != nil {
				cpuFound.Estado = true
				p.CPUSemaphore <- struct{}{}
			}
			break
		}
	}

	// Si no está en EXEC, buscar en BLOCKED
	if proceso == nil {
		p.mutexBlockQueue.Lock()
		for i, proc := range p.Planificador.BlockQueue {
			if proc.PCB.PID == pid {
				proceso = proc
				estadoAnterior = "BLOCKED"
				// Sacarlo de BLOCKED
				p.Planificador.BlockQueue = append(p.Planificador.BlockQueue[:i], p.Planificador.BlockQueue[i+1:]...)
				break
			}
		}
		p.mutexBlockQueue.Unlock()
	}

	// Si no está en BLOCKED, buscar en SUSP.BLOCKED
	if proceso == nil {
		p.mutexSuspBlockQueue.Lock()
		for i, proc := range p.Planificador.SuspBlockQueue {
			if proc.PCB.PID == pid {
				proceso = proc
				estadoAnterior = "SUSP.BLOCKED"
				// Sacarlo de SUSP.BLOCKED
				p.Planificador.SuspBlockQueue = append(p.Planificador.SuspBlockQueue[:i], p.Planificador.SuspBlockQueue[i+1:]...)
				break
			}
		}
		p.mutexSuspBlockQueue.Unlock()
	}

	// Si no está en ninguna de las anteriores, buscar en READY
	if proceso == nil {
		p.mutexReadyQueue.Lock()
		for i, proc := range p.Planificador.ReadyQueue {
			if proc.PCB.PID == pid {
				proceso = proc
				estadoAnterior = "READY"
				// Sacarlo de READY
				p.Planificador.ReadyQueue = append(p.Planificador.ReadyQueue[:i], p.Planificador.ReadyQueue[i+1:]...)
				break
			}
		}
		p.mutexReadyQueue.Unlock()
	}

	// Si no está en READY, buscar en SUSP.READY
	if proceso == nil {
		p.mutexSuspReadyQueue.Lock()
		for i, proc := range p.Planificador.SuspReadyQueue {
			if proc.PCB.PID == pid {
				proceso = proc
				estadoAnterior = "SUSP.READY"
				// Sacarlo de SUSP.READY
				p.Planificador.SuspReadyQueue = append(p.Planificador.SuspReadyQueue[:i], p.Planificador.SuspReadyQueue[i+1:]...)
				break
			}
		}
		p.mutexSuspReadyQueue.Unlock()
	}

	if proceso == nil {
		p.Log.Error("No se encontró el proceso en ninguna cola",
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

	// 3. Actualizar métricas según el estado anterior
	proceso.PCB.MetricasEstado[internal.EstadoExit]++
	if proceso.PCB.MetricasTiempo[internal.EstadoExit] == nil {
		proceso.PCB.MetricasTiempo[internal.EstadoExit] = &internal.EstadoTiempo{}
	}
	proceso.PCB.MetricasTiempo[internal.EstadoExit].TiempoInicio = time.Now()

	// Actualizar tiempo acumulado del estado anterior
	switch estadoAnterior {
	case "EXEC":
		if proceso.PCB.MetricasTiempo[internal.EstadoExec] != nil {
			proceso.PCB.MetricasTiempo[internal.EstadoExec].TiempoAcumulado += time.Since(proceso.PCB.MetricasTiempo[internal.EstadoExec].TiempoInicio)
		}
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
	p.Log.Info(fmt.Sprintf("## (%d) - Métricas de estado", proceso.PCB.PID),
		log.AnyAttr("metricas_estado", proceso.PCB.MetricasEstado),
		log.AnyAttr("metricas_tiempo", proceso.PCB.MetricasTiempo),
	)

	// 4. Liberar PCB
	proceso.PCB = nil

	// 5. Checkear si hay procesos suspendidos que puedan volver a memoria
	p.CheckearEspacioEnMemoria()
}
