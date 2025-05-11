package planificadores

import (
	"bufio"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/sisoputnfrba/tp-golang/kernel/internal"
	"github.com/sisoputnfrba/tp-golang/utils/log"
)

const (
	PlanificadorEstadoStop  = "STOP"
	PlanificadorEstadoStart = "START"
)

// TODO: Agregar escucha ante un nuevo proceso en la cola de New y ante un enter.

// PlanificadorLargoPlazoFIFO realiza las funciones correspondientes al planificador de largo plazo FIFO.
func (p *Service) PlanificadorLargoPlazoFIFO() {
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
	//estado = PlanificadorEstadoStart
	p.Log.Info("Planificador de largo plazo iniciado")

	if estado == PlanificadorEstadoStart {
		for _, proceso := range p.Planificador.SuspReadyQueue {
			if p.Memoria.ConsultarEspacio() {
				// Si el proceso se carga en memoria, lo muevo a la cola de ready
				// y lo elimino de la cola de suspendidos ready

				p.Planificador.SuspReadyQueue = p.Planificador.SuspReadyQueue[1:] // lo saco de la cola
				timeSusp := proceso.PCB.MetricasTiempo[internal.EstadoSuspReady]
				timeSusp.TiempoAcumulado = timeSusp.TiempoAcumulado + time.Since(timeSusp.TiempoInicio)

				// Agrego el proceso a la cola de ready
				p.Planificador.ReadyQueue = append(p.Planificador.ReadyQueue, proceso)
				if proceso.PCB.MetricasTiempo[internal.EstadoReady] == nil {
					proceso.PCB.MetricasTiempo[internal.EstadoReady] = &internal.EstadoTiempo{}
				}
				proceso.PCB.MetricasTiempo[internal.EstadoReady].TiempoInicio = time.Now()

				proceso.PCB.MetricasEstado[internal.EstadoReady]++

				p.Log.Info(fmt.Sprintf("%d Pasa del estado SUSP.READY al estado READY", proceso.PCB.PID))
			} else {
				/* Si la respuesta es negativa (ya que la Memoria no tiene espacio suficiente para inicializarlo)
				se deberá esperar la finalización de otro proceso para volver a intentar inicializarlo.
				Vuelvo a agregar al proceso a la cola de suspendidos ready en el lugar que estaba (al principio por ser FIFO) */
				p.Planificador.SuspReadyQueue = append([]*internal.Proceso{proceso}, p.Planificador.SuspReadyQueue...)
			}
		}

		for _, proceso := range p.Planificador.NewQueue {
			if p.Memoria.ConsultarEspacio() {
				// Si el proceso se carga en memoria, lo muevo a la cola de ready
				// y lo elimino de la cola de new

				p.Planificador.NewQueue = p.Planificador.NewQueue[1:] // lo saco de la cola
				timeNew := proceso.PCB.MetricasTiempo[internal.EstadoNew]
				timeNew.TiempoAcumulado = timeNew.TiempoAcumulado + time.Since(timeNew.TiempoInicio)

				// Agrego el proceso a la cola de ready
				p.Planificador.ReadyQueue = append(p.Planificador.ReadyQueue, proceso)
				if proceso.PCB.MetricasTiempo[internal.EstadoReady] == nil {
					proceso.PCB.MetricasTiempo[internal.EstadoReady] = &internal.EstadoTiempo{}
				}
				proceso.PCB.MetricasTiempo[internal.EstadoReady].TiempoInicio = time.Now()
				proceso.PCB.MetricasEstado[internal.EstadoReady]++

				p.Log.Info(fmt.Sprintf("%d Pasa del estado NEW al estado READY", proceso.PCB.PID))
				// proceso.PCB = nil // Libero el PCB asociado al proceso
			} else {
				/* Si la respuesta es negativa (ya que la Memoria no tiene espacio suficiente para inicializarlo)
				se deberá esperar la finalización de otro proceso para volver a intentar inicializarlo.
				Vuelvo a agregar al proceso a la cola de new en el lugar que estaba (al principio por ser FIFO) */
				p.Planificador.NewQueue = append([]*internal.Proceso{proceso}, p.Planificador.NewQueue...)
			}
		}
	}
}

func (p *Service) FinalizarProceso(proceso internal.Proceso) {
	// 1. Notificar a Memoria
	status, err := p.Memoria.FinalizarProceso(proceso.PCB.PID)
	if err != nil || status != http.StatusOK {
		p.Log.Error("Error al finalizar proceso en memoria",
			log.ErrAttr(err),
			log.IntAttr("PID", proceso.PCB.PID),
		)
		return
	}

	// 2. Loguear métricas (acá deberías tenerlas guardadas en el PCB)
	p.Log.Info("Finaliza el proceso", log.IntAttr("PID", proceso.PCB.PID))
	p.Log.Info("Métricas de estado",
		log.AnyAttr("metricas_estado", proceso.PCB.MetricasEstado),
		log.AnyAttr("metricas_tiempo", proceso.PCB.MetricasTiempo),
	)

	// 3. Liberar PCB
	// Asumiendo que mantenés un map[PID]PCB
	//delete(pcbTable, proceso.ID)
}
