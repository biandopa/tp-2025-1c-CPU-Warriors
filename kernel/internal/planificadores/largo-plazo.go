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
	//	p.Planificador.NewQueue = append([]*internal.Proceso{proceso}, p.Planificador.NewQueue...)
	p.Planificador.NewQueue = append(p.Planificador.NewQueue, proceso)
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
	p.mutexSuspReadyQueue.Lock()
	i := 0
	for i < len(p.Planificador.SuspReadyQueue) {
		proceso := p.Planificador.SuspReadyQueue[i]
		if p.Memoria.ConsultarEspacio(proceso.PCB.Tamanio, proceso.PCB.PID) {
			// Si el proceso se carga en memoria, lo muevo a la cola de ready
			// y lo elimino de la cola de suspendidos ready

			// Remover el proceso de la cola
			p.Planificador.SuspReadyQueue, _ = p.removerDeCola(proceso.PCB.PID, p.Planificador.SuspReadyQueue)

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
	//p.mutexSuspReadyQueue.Unlock()

	if len(p.Planificador.SuspReadyQueue) == 0 {
		p.mutexNewQueue.Lock()
		i := 0
		for i < len(p.Planificador.NewQueue) {
			proceso := p.Planificador.NewQueue[i]
			if p.Memoria.ConsultarEspacio(proceso.PCB.Tamanio, proceso.PCB.PID) {
				// Si el proceso se carga en memoria, lo muevo a la cola de ready
				// y lo elimino de la cola de new

				// Remover el proceso de la cola usando índice
				p.Planificador.NewQueue, _ = p.removerDeCola(proceso.PCB.PID, p.Planificador.NewQueue)

				if proceso.PCB.MetricasTiempo[internal.EstadoNew] == nil {
					proceso.PCB.MetricasTiempo[internal.EstadoNew] = &internal.EstadoTiempo{}
				}
				timeNew := proceso.PCB.MetricasTiempo[internal.EstadoNew]
				timeNew.TiempoAcumulado = timeNew.TiempoAcumulado + time.Since(timeNew.TiempoInicio)

				if proceso.PCB.MetricasTiempo[internal.EstadoReady] == nil {
					proceso.PCB.MetricasTiempo[internal.EstadoReady] = &internal.EstadoTiempo{}
				}

				p.Memoria.CargarProcesoEnMemoriaDeSistema(proceso.PCB.NombreArchivo, proceso.PCB.PID)

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
	p.mutexSuspReadyQueue.Unlock()
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
	for i, proc := range p.Planificador.ExecQueue {
		if proc.PCB.PID == pid {
			// Sacar de la cola de exec
			p.Planificador.ExecQueue = append(p.Planificador.ExecQueue[:i], p.Planificador.ExecQueue[i+1:]...)
			break
		}
		proc.PCB.MetricasTiempo[internal.EstadoExec].TiempoAcumulado +=
			time.Since(proceso.PCB.MetricasTiempo[internal.EstadoExec].TiempoInicio)
	}
	p.mutexExecQueue.Unlock()

	// 4. DESPUÉS: Actualizar ráfaga anterior (IMPORTANTE para SRT)
	/*p.actualizarRafagaAnterior(proceso)

	// 5. Cambiar el estado de la CPU y notificar al planificador
	cpuFound := p.buscarCPUPorPID(proceso.PCB.PID)
	if cpuFound != nil {
		p.LiberarCPU(cpuFound) // Usar función centralizada que incluye notificación al planificador
	}*/

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

// FinalizarProcesoEnCualquierCola busca un proceso en todas las colas y lo finaliza.
func (p *Service) FinalizarProcesoEnCualquierCola(pid int) {
	proceso, cola := p.BuscarProcesoEnCualquierCola(pid)

	// Vovler a revisar si estaba en otra cola más
	if proceso2, cola2 := p.BuscarProcesoEnCualquierCola(pid); proceso2 != nil && proceso2.PCB.PID == pid {
		proceso = proceso2
		cola = cola2
	}

	// Si encontré proceso en EXEC, actualizar y liberar CPU
	if proceso != nil {
		// Actualizar métricas de tiempo
		if proceso.PCB.MetricasTiempo[cola] != nil {
			proceso.PCB.MetricasTiempo[cola].TiempoAcumulado +=
				time.Since(proceso.PCB.MetricasTiempo[cola].TiempoInicio)
		}

		switch cola {
		case internal.EstadoExec:
			// Liberar CPU
			/*cpuFound := p.buscarCPUPorPID(proceso.PCB.PID)
			if cpuFound != nil {
				p.LiberarCPU(cpuFound)
			}*/

			p.mutexExecQueue.Lock()
			p.Planificador.ExecQueue, _ = p.removerDeCola(pid, p.Planificador.ExecQueue)
			p.mutexExecQueue.Unlock()
		case internal.EstadoReady:
			p.mutexReadyQueue.Lock()
			p.Planificador.ReadyQueue, _ = p.removerDeCola(pid, p.Planificador.ReadyQueue)
			p.mutexReadyQueue.Unlock()
		case internal.EstadoBloqueado:
			p.mutexBlockQueue.Lock()
			p.Planificador.BlockQueue, _ = p.removerDeCola(pid, p.Planificador.BlockQueue)
			p.mutexBlockQueue.Unlock()
		case internal.EstadoSuspBloqueado:
			p.mutexSuspBlockQueue.Lock()
			p.Planificador.SuspBlockQueue, _ = p.removerDeCola(pid, p.Planificador.SuspBlockQueue)
			p.mutexSuspBlockQueue.Unlock()
		case internal.EstadoSuspReady:
			p.mutexSuspReadyQueue.Lock()
			p.Planificador.SuspReadyQueue, _ = p.removerDeCola(pid, p.Planificador.SuspReadyQueue)
			p.mutexSuspReadyQueue.Unlock()
		case internal.EstadoNew:
			p.Planificador.NewQueue, _ = p.removerDeCola(pid, p.Planificador.NewQueue)
		default:
			p.Log.Error("🚨 Estado no reconocido al finalizar proceso",
				log.IntAttr("pid", pid),
				log.StringAttr("estado", string(cola)),
			)
		}
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

	// 3. Actualizar métricas de Exit
	proceso.PCB.MetricasEstado[internal.EstadoExit]++
	if proceso.PCB.MetricasTiempo[internal.EstadoExit] == nil {
		proceso.PCB.MetricasTiempo[internal.EstadoExit] = &internal.EstadoTiempo{}
	}
	proceso.PCB.MetricasTiempo[internal.EstadoExit].TiempoInicio = time.Now()

	//Log obligatorio: Cambio de estado
	p.Log.Info(fmt.Sprintf("## (%d) Pasa del estado %s al estado EXIT", proceso.PCB.PID, cola))

	//Log obligatorio: Finalización de proceso
	p.Log.Info(fmt.Sprintf("## (%d) Finaliza el proceso", proceso.PCB.PID))

	proceso.PCB.MetricasTiempo[internal.EstadoExit].TiempoAcumulado +=
		time.Since(proceso.PCB.MetricasTiempo[internal.EstadoExit].TiempoInicio)

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

	proceso.PCB = nil // Liberar referencia al proceso
	proceso = nil     // Liberar referencia al proceso

	// 4. Checkear si hay procesos suspendidos que puedan volver a memoria
	p.CheckearEspacioEnMemoria()
}
