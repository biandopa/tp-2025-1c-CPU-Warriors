package planificadores

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/sisoputnfrba/tp-golang/kernel/internal"
	"github.com/sisoputnfrba/tp-golang/utils/log"
)

// TODO: Agregar escucha ante un nuevo proceso en la cola de ready
func (p *Service) PlanificadorCortoPlazoFIFO() {

	for _, proceso := range p.Planificador.ReadyQueue {

		var cpuSeleccionada *CpuIdentificacion
		for {
			if len(p.CPUConectadas) > 0 {
				for i := range p.CPUConectadas {
					if p.CPUConectadas[i].ESTADO {

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
						cpuSeleccionada = &p.CPUConectadas[i]
						p.CPUConectadas[i].ESTADO = false
						fmt.Println("CPU seleccionada:", cpuSeleccionada)

						p.enviarProcesoACPU(*cpuSeleccionada, proceso) // Envio el proceso a la CPU seleccionada
						// Se encontró una CPU activa, salir de la función y mandarle la cpu y el PID Y PC
					}
				}
			}
		}
	}
}

func (p *Service) enviarProcesoACPU(cpuID CpuIdentificacion, proceso *internal.Proceso) {

	p.Log.Debug("Entre al EjecutarPlanificadores")
	data := map[string]interface{}{
		"cpuID": cpuID,
		"pc":    proceso.PCB.PID,
		"pid":   proceso.PCB.ProgramCounter, // Cambiar por el ID real
	}

	body, err := json.Marshal(data)
	if err != nil {
		p.Log.Error("Error al serializar ioIdentificacion",
			slog.Attr{Key: "error", Value: slog.StringValue(err.Error())},
		)
		return
	}

	url := fmt.Sprintf("http://%s:%d/kernel/procesos", cpuID.IP, cpuID.Puerto)
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(body))
	if err != nil {
		p.Log.Error("error enviando mensaje",
			slog.Attr{Key: "error", Value: slog.StringValue(err.Error())},
			slog.Attr{Key: "ip", Value: slog.StringValue(cpuID.IP)},
			slog.Attr{Key: "puerto", Value: slog.IntValue(cpuID.Puerto)},
		)
	}

	if resp != nil {
		p.Log.Info("Respuesta del servidor",
			slog.Attr{Key: "status", Value: slog.StringValue(resp.Status)},
			slog.Attr{Key: "body", Value: slog.StringValue(string(body))},
		)
	} else {
		p.Log.Info("Respuesta del servidor: nil")
	}
}
