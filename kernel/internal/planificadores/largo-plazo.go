package planificadores

import (
	"log/slog"
	"time"

	"github.com/sisoputnfrba/tp-golang/kernel/internal"
	"github.com/sisoputnfrba/tp-golang/kernel/pkg/memoria"
	"github.com/sisoputnfrba/tp-golang/utils/log"
)

const (
	PlanificadorEstadoStop  = "STOP"
	PlanificadorEstadoStart = "START"
)

type Service struct {
	Planificador *Planificador
	Log          *slog.Logger
	Memoria      *memoria.Memoria
}

type Planificador struct {
	NewQueue       []*internal.Proceso
	ReadyQueue     []*internal.Proceso
	BlockQueue     []*internal.Proceso
	SuspReadyQueue []*internal.Proceso
	SuspBlockQueue []*internal.Proceso
	ExecQueue      []*internal.Proceso
	ExitQueue      []*internal.Proceso
}

func NewPlanificador(log *slog.Logger, ipMemoria string, puertoMemoria int) *Service {
	return &Service{
		Planificador: &Planificador{
			NewQueue:       make([]*internal.Proceso, 0),
			ReadyQueue:     make([]*internal.Proceso, 0),
			BlockQueue:     make([]*internal.Proceso, 0),
			SuspReadyQueue: make([]*internal.Proceso, 0),
			SuspBlockQueue: make([]*internal.Proceso, 0),
			ExecQueue:      make([]*internal.Proceso, 0),
			ExitQueue:      make([]*internal.Proceso, 0),
		},
		Log:     log,
		Memoria: memoria.NewMemoria(ipMemoria, puertoMemoria),
	}
}

// Planificador de largo plazo FIFO
func (p *Service) PlanificadorLargoPlazoFIFO(enter string) {
	estado := PlanificadorEstadoStop

	// Revisa si el argumento es "Enter" para iniciar el planificador
	if enter == "\n" {
		estado = PlanificadorEstadoStart
	}

	if estado == PlanificadorEstadoStart {
		for _, proceso := range p.Planificador.SuspReadyQueue {
			// TODO: Implementar funcion de verificación de memoria
			if memoria.IntentarCargarEnMemoria(proceso) {
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

				p.Log.Info("Proceso movido de SUSP.READY a READY",
					log.IntAttr("PID", proceso.PCB.PID),
				)
			} else {
				// Me quedo escuchando la respuesta de la memoria ante la finalización de un proceso
				// TODO: Implementar la función de escucha
			}

		}

		for _, proceso := range p.Planificador.NewQueue {
			if memoria.IntentarCargarEnMemoria(proceso) {
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

				p.Log.Info("Proceso movido de NEW a READY",
					log.IntAttr("PID", proceso.PCB.PID),
				)
				// proceso.PCB = nil // Libero el PCB asociado al proceso
			} else {
				// Me quedo escuchando la respuesta de la memoria ante la finalización de un proceso
				// TODO: Implementar la función de escucha
			}
		}
		time.Sleep(time.Second) // espera mínima para no sobrecargar CPU
	}
}

/*func (p *Service) FinalizarProceso(proceso internal.Proceso) {
	// 1. Notificar a Memoria
	url := fmt.Sprintf("http://%s:%d/memoria/finalizar-proceso", h.Config.IpMemory, h.Config.PortMemory) // TODO: Hacer la llamada en el pkg memoria

	body, _ := json.Marshal(map[string]int{"pid": proceso.PCB.PID})

	resp, err := http.Post(url, "application/json", bytes.NewBuffer(body))
	if err != nil || resp.StatusCode != http.StatusOK {
		p.Log.Error("Fallo al finalizar proceso en Memoria",
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

	// 4. Intentar inicializar procesos esperando
	//p.planilargoplazofifo()
}*/
