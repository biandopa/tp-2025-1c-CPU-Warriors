package planificadores

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/sisoputnfrba/tp-golang/kernel/internal"
	"github.com/sisoputnfrba/tp-golang/utils/log"
)

type Usleep struct {
	PID         int `json:"pid"`
	TiempoSleep int `json:"tiempo_sleep"`
}

// EnviarUsleep envia un usleep al IO
func (p *Service) EnviarUsleep(puertoIO int, iPIO string, pid, timeSleep int) {
	// Crear el JSON con los datos necesarios
	usleep := &Usleep{
		PID:         pid,
		TiempoSleep: timeSleep,
	}

	jsonData, err := json.Marshal(usleep)
	if err != nil {
		p.Log.Error("Error al serializar el usleep a JSON",
			log.ErrAttr(err),
			log.IntAttr("pid", pid),
		)
		return
	}

	// Realizar la petición POST al IO
	url := fmt.Sprintf("http://%s:%d/kernel/usleep", iPIO, puertoIO)
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		p.Log.Error("Error al enviar el usleep al IO",
			log.ErrAttr(err),
			log.IntAttr("pid", pid),
		)
		return
	}

	if resp != nil {
		defer func() {
			if err = resp.Body.Close(); err != nil {
				fmt.Println("Error cerrando el cuerpo de la respuesta:", err)
			}
		}()
	}

	p.Log.Debug("Respuesta del IO al usleep",
		log.IntAttr("status_code", resp.StatusCode),
		log.AnyAttr("response_body", resp.Body),
	)
}

// BloquearPorIO mueve un proceso de EXEC a BLOCKED por una operación de IO
func (p *Service) BloquearPorIO(pid int) error {
	// Buscar el proceso en la cola de EXEC
	var proceso *internal.Proceso

	p.mutexExecQueue.Lock()
	for i, proc := range p.Planificador.ExecQueue {
		if proc.PCB.PID == pid {
			proceso = proc
			// Remover de EXEC
			p.Planificador.ExecQueue = append(p.Planificador.ExecQueue[:i], p.Planificador.ExecQueue[i+1:]...)
			break
		}
	}
	p.mutexExecQueue.Unlock()

	if proceso == nil {
		return fmt.Errorf("proceso con PID %d no encontrado en EXEC", pid)
	}

	// Actualizar métricas de tiempo para EXEC
	if proceso.PCB.MetricasTiempo[internal.EstadoExec] != nil {
		proceso.PCB.MetricasTiempo[internal.EstadoExec].TiempoAcumulado +=
			time.Since(proceso.PCB.MetricasTiempo[internal.EstadoExec].TiempoInicio)
	}

	// Agregar a BLOCKED
	p.mutexBlockQueue.Lock()
	p.Planificador.BlockQueue = append(p.Planificador.BlockQueue, proceso)
	p.mutexBlockQueue.Unlock()

	// Inicializar métricas de tiempo para BLOCKED
	if proceso.PCB.MetricasTiempo[internal.EstadoBloqueado] == nil {
		proceso.PCB.MetricasTiempo[internal.EstadoBloqueado] = &internal.EstadoTiempo{}
	}
	proceso.PCB.MetricasTiempo[internal.EstadoBloqueado].TiempoInicio = time.Now()
	proceso.PCB.MetricasEstado[internal.EstadoBloqueado]++

	// Notificar al planificador de mediano plazo
	p.CanalNuevoProcBlocked <- proceso

	return nil
}
