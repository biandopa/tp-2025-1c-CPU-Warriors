package planificadores

import (
	"fmt"
	"time"

	"github.com/sisoputnfrba/tp-golang/kernel/internal"
	"github.com/sisoputnfrba/tp-golang/utils/log"
)

func (p *Service) PlanificadorCortoPlazoFIFO(enter string, proceso string) {

	for _, proceso := range p.Planificador.ReadyQueue {

		var cpuSeleccionada *CPUIdentificacion
		for {
			if len(h.CPUConectadas) > 0 {
				for i := range h.CPUConectadas {
					if h.CPUConectadas[i].ESTADO {

						// Si el proceso ouede ejecutarse en una cpu, lo muevo a la cola de EXEC
						// y lo elimino de la cola de Ready
						p.Planificador.ReadyQueue = p.Planificador.ReadyQueue[1:] // lo saco de la cola
						timeNew := proceso.PCB.MetricasTiempo[internal.EstadoReady]
						timeNew.TiempoAcumulado = timeNew.TiempoAcumulado + time.Since(timeNew.TiempoInicio)

						// Agrego el proceso a la cola de EXEC
						p.Planificador.ExecQueue = append(p.Planificador.ExecQueue, proceso)
						if proceso.PCB.MetricasTiempo[internal.EstadoExec] == nil {
							proceso.PCB.MetricasTiempo[internal.EstadoExec] = &internal.EstadoTiempo{}
						}
						proceso.PCB.MetricasTiempo[internal.EstadoExec].TiempoInicio = time.Now()

						proceso.PCB.MetricasEstado[internal.EstadoExec]++

						p.Log.Info("Proceso movido de READY a EXEC",
							log.IntAttr("PID", proceso.PCB.PID),
						)
						cpuSeleccionada = &h.CPUConectadas[i]
						h.CPUConectadas[i].ESTADO = false
						fmt.Println("CPU seleccionada:", cpuSeleccionada)
						return // Se encontró una CPU activa, salir de la función y mandarle la cpu y el PID Y PC
					}
				}
			}
		}
	}
}
