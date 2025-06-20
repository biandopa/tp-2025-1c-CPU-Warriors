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

// PlanificadorLargoPlazoFIFO realiza las funciones correspondientes al planificador de largo plazo FIFO.
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

			// Checkear si la cola de suspReady puede ingresar, en caso de que se vacie consultar la de NEW

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

			p.Planificador.NewQueue = append(
				p.Planificador.NewQueue[:i+1],  // lo que viene después
				p.Planificador.NewQueue[i:]..., // desplazamos lo que estaba en i
			)
			p.Planificador.NewQueue[i] = proceso

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

	for _, proceso := range p.Planificador.SuspReadyQueue {
		if p.Memoria.ConsultarEspacio(proceso.PCB.NombreArchivo, proceso.PCB.Tamanio, proceso.PCB.PID) {
			// Si el proceso se carga en memoria, lo muevo a la cola de ready
			// y lo elimino de la cola de suspendidos ready

			p.Planificador.SuspReadyQueue = p.Planificador.SuspReadyQueue[1:] // lo saco de la cola
			if proceso.PCB.MetricasTiempo[internal.EstadoSuspReady] == nil {
				proceso.PCB.MetricasTiempo[internal.EstadoSuspReady] = &internal.EstadoTiempo{}
			}
			timeSusp := proceso.PCB.MetricasTiempo[internal.EstadoSuspReady]
			timeSusp.TiempoAcumulado = timeSusp.TiempoAcumulado + time.Since(timeSusp.TiempoInicio)

			// Agrego el proceso a la cola de ready
			p.Planificador.ReadyQueue = append(p.Planificador.ReadyQueue, proceso)
			if len(p.canalNuevoProcesoReady) == 0 {
				//p.canalNuevoProcesoReady <- struct{}{}
			}

			if proceso.PCB.MetricasTiempo[internal.EstadoReady] == nil {
				proceso.PCB.MetricasTiempo[internal.EstadoReady] = &internal.EstadoTiempo{}
			}
			proceso.PCB.MetricasTiempo[internal.EstadoReady].TiempoInicio = time.Now()

			proceso.PCB.MetricasEstado[internal.EstadoReady]++

			p.Log.Info(fmt.Sprintf("%d Pasa del estado SUSP.READY al estado READY", proceso.PCB.PID))
		} else {
			break
		}
	}

	if len(p.Planificador.SuspReadyQueue) != 0 {
		for _, proceso := range p.Planificador.NewQueue {
			if p.Memoria.ConsultarEspacio(proceso.PCB.NombreArchivo, proceso.PCB.Tamanio, proceso.PCB.PID) {
				// Si el proceso se carga en memoria, lo muevo a la cola de ready
				// y lo elimino de la cola de new

				p.mutexNewQueue.Lock()
				p.Planificador.NewQueue = p.Planificador.NewQueue[1:] // lo saco de la cola
				p.mutexNewQueue.Unlock()

				if proceso.PCB.MetricasTiempo[internal.EstadoNew] == nil {
					proceso.PCB.MetricasTiempo[internal.EstadoNew] = &internal.EstadoTiempo{}
				}
				timeNew := proceso.PCB.MetricasTiempo[internal.EstadoNew]
				timeNew.TiempoAcumulado = timeNew.TiempoAcumulado + time.Since(timeNew.TiempoInicio)

				// Notificar al channel de nuevo proceso ready
				p.canalNuevoProcesoReady <- proceso

				if proceso.PCB.MetricasTiempo[internal.EstadoReady] == nil {
					proceso.PCB.MetricasTiempo[internal.EstadoReady] = &internal.EstadoTiempo{}
				}
				proceso.PCB.MetricasTiempo[internal.EstadoReady].TiempoInicio = time.Now()
				proceso.PCB.MetricasEstado[internal.EstadoReady]++

				p.Log.Info(fmt.Sprintf("%d Pasa del estado NEW al estado READY", proceso.PCB.PID))
			} else {
				break
			}
		}
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
	// TODO: Preguntar el sabado :)

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
	p.Log.Info("Finaliza el proceso", log.IntAttr("PID", proceso.PCB.PID))
	p.Log.Info("Métricas de estado",
		log.AnyAttr("metricas_estado", proceso.PCB.MetricasEstado),
		log.AnyAttr("metricas_tiempo", proceso.PCB.MetricasTiempo),
	)

	// 7. Liberar PCB
	proceso.PCB = nil // Libero el PCB asociado al proceso

	// 8. Le avisamos al channel de nuevo proceso ready
	p.CheckearEspacioEnMemoria()
}
