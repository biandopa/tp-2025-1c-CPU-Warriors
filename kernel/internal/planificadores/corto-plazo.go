package planificadores

import (
	"fmt"
	"time"

	"github.com/sisoputnfrba/tp-golang/kernel/internal"
	"github.com/sisoputnfrba/tp-golang/kernel/pkg/cpu"
	"github.com/sisoputnfrba/tp-golang/utils/log"
)

func (p *Service) PlanificadorCortoPlazoFIFO() {
	go func() {
		for {
			proceso := <-p.canalNuevoProcesoReady // Espera una notificación

			for len(p.Planificador.ReadyQueue) > 0 { // Procesa mientras haya elementos en ReadyQueue

				var cpuSeleccionada *cpu.Cpu
				for {
					if len(p.CPUsConectadas) > 0 {
						for i := range p.CPUsConectadas {
							if p.CPUsConectadas[i].Estado {
								// Mover proceso de READY a EXEC
								p.Planificador.ReadyQueue = p.Planificador.ReadyQueue[1:]
								timeNew := proceso.PCB.MetricasTiempo[internal.EstadoReady]
								timeNew.TiempoAcumulado += time.Since(timeNew.TiempoInicio)

								p.Planificador.ExecQueue = append(p.Planificador.ExecQueue, proceso)
								if proceso.PCB.MetricasTiempo[internal.EstadoExec] == nil {
									proceso.PCB.MetricasTiempo[internal.EstadoExec] = &internal.EstadoTiempo{}
								}
								proceso.PCB.MetricasTiempo[internal.EstadoExec].TiempoInicio = time.Now()
								proceso.PCB.MetricasEstado[internal.EstadoExec]++

								p.Log.Info("Proceso movido de READY a EXEC",
									log.IntAttr("PID", proceso.PCB.PID),
								)
								cpuSeleccionada = p.CPUsConectadas[i]
								p.CPUsConectadas[i].Estado = false
								fmt.Println("CPU seleccionada:", cpuSeleccionada)

								cpuSeleccionada.DispatchProcess(proceso.PCB.PID, proceso.PCB.PC)
								break
							}
						}
					}
					if cpuSeleccionada != nil {
						break
					}
				}
			}
		}
	}()
}
