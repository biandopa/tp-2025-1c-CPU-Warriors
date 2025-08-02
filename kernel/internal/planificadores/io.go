package planificadores

import (
	"bytes"
	"encoding/json"
	"fmt"
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
	resp, err := p.HttpClient.Post(url, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		p.Log.Debug("Error al enviar el usleep al IO",
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
	for _, proc := range p.Planificador.ExecQueue {
		if proc != nil && proc.PCB != nil && proc.PCB.PID == pid {
			proceso = proc
			break
		}
	}

	// Remover de EXEC usando función segura
	if proceso != nil {
		var removido bool
		p.Planificador.ExecQueue, removido = p.removerDeCola(pid, p.Planificador.ExecQueue)
		if removido {
			proceso.PCB.MetricasTiempo[internal.EstadoExec].TiempoAcumulado +=
				time.Since(proceso.PCB.MetricasTiempo[internal.EstadoExec].TiempoInicio)
		}
	}

	p.mutexExecQueue.Unlock()

	if proceso == nil {
		//p.mutexExecQueue.Unlock()
		return fmt.Errorf("proceso con PID %d no encontrado en EXEC", pid)
	}

	// Actualizar ráfaga anterior antes de bloquear (IMPORTANTE para SRT - incluye métricas de tiempo EXEC)
	//p.actualizarRafagaAnterior(proceso)

	// Agregar a BLOCKED
	p.mutexBlockQueue.Lock()
	p.Planificador.BlockQueue = append(p.Planificador.BlockQueue, proceso)

	//Log obligatorio: Cambio de estado
	// "## (<PID>) Pasa del estado <ESTADO_ANTERIOR> al estado <ESTADO_ACTUAL>"
	p.Log.Info(fmt.Sprintf("## (%d) Pasa del estado EXEC al estado BLOCKED", pid))

	// Inicializar métricas de tiempo para BLOCKED
	if proceso.PCB.MetricasTiempo[internal.EstadoBloqueado] == nil {
		proceso.PCB.MetricasTiempo[internal.EstadoBloqueado] = &internal.EstadoTiempo{}
	}
	proceso.PCB.MetricasTiempo[internal.EstadoBloqueado].TiempoInicio = time.Now()
	proceso.PCB.MetricasEstado[internal.EstadoBloqueado]++
	p.mutexBlockQueue.Unlock()

	// Notificar al planificador de mediano plazo
	p.CanalNuevoProcBlocked <- proceso

	return nil
}
