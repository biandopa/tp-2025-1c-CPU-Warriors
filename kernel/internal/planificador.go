package internal

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"time"
)

type Planificador struct {
	NewQueue       []*Proceso
	ReadyQueue     []*Proceso
	BlockQueue     []*Proceso
	SuspReadyQueue []*Proceso
	SuspBlockQueue []*Proceso
	ExecQueue      []*Proceso
	ExitQueue      []*Proceso
}

// Planificador de largo plazo FIFO
func (p *Planificador) PlanificadorLargoPlazoFIFO(enter string) {
	estado := "STOP"

	// Revisa si el argumento es "Enter" para iniciar el planificador
	if enter == "\n" {
		estado = "START"
	}

	if estado == "START" {
		for _, proceso := range p.SuspReadyQueue {
			// TODO: Implementar funcion de verificación de memoria
			if h.IntentarCargarEnMemoria(proceso) {
				// Si el proceso se carga en memoria, lo muevo a la cola de ready
				// y lo elimino de la cola de suspendidos ready

				p.SuspReadyQueue = p.SuspReadyQueue[1:] // lo saco de la cola
				timeSusp := proceso.PCB.MetricasTiempo["SUSPENDIDO"]
				timeSusp.TiempoAcumulado = timeSusp.TiempoAcumulado + time.Since(timeSusp.TiempoInicio)

				// Agrego el proceso a la cola de ready
				p.ReadyQueue = append(p.ReadyQueue, proceso)
				proceso.PCB.MetricasTiempo["READY"].TiempoInicio = time.Now()
				proceso.PCB.MetricasEstado["READY"]++

				h.Log.Info("Proceso movido de SUSP.READY a READY", slog.Int("PID", proceso.ID))
			} else {
				// Me quedo escuchando la respuesta de la memoria ante la finalización de un proceso
				// TODO: Implementar la función de escucha
			}

		}

		for _, proceso := range p.NewQueue {
			if h.IntentarCargarEnMemoria(proceso) {
				// Si el proceso se carga en memoria, lo muevo a la cola de ready
				// y lo elimino de la cola de new

				p.NewQueue = p.NewQueue[1:] // lo saco de la cola
				timeNew := proceso.PCB.MetricasTiempo["NEW"]
				timeNew.TiempoAcumulado = timeNew.TiempoAcumulado + time.Since(timeNew.TiempoInicio)

				// Agrego el proceso a la cola de ready
				p.ReadyQueue = append(p.ReadyQueue, proceso)
				proceso.PCB.MetricasTiempo["READY"].TiempoInicio = time.Now()
				proceso.PCB.MetricasEstado["READY"]++

				h.Log.Info("Proceso movido de NEW a READY", slog.Int("PID", proceso.ID))
				// proceso.PCB = nil // Libero el PCB asociado al proceso
			} else {
				// Me quedo escuchando la respuesta de la memoria ante la finalización de un proceso
				// TODO: Implementar la función de escucha
			}
		}
		time.Sleep(time.Second) // espera mínima para no sobrecargar CPU
	}
}

func (p *Planificador) FinalizarProceso(proceso Proceso) {
	// 1. Notificar a Memoria
	url := fmt.Sprintf("http://%s:%d/memoria/finalizar-proceso", h.Config.IpMemory, h.Config.PortMemory)

	body, _ := json.Marshal(map[string]int{"pid": proceso.ID})

	resp, err := http.Post(url, "application/json", bytes.NewBuffer(body))
	if err != nil || resp.StatusCode != http.StatusOK {
		h.Log.Error("Fallo al finalizar proceso en Memoria", slog.Int("PID", proceso.ID))
		return
	}

	// 2. Loguear métricas (acá deberías tenerlas guardadas en el PCB)
	h.Log.Info(fmt.Sprintf("## %d ⏹ Finaliza el proceso", proceso.ID))
	h.Log.Info(fmt.Sprintf("## %d 📊 Métricas de estado: NEW (%d veces, %d ms), READY (...), EXEC (...)", proceso.ID, proceso.Metricas.NEWCount, proceso.Metricas.NEWTime))

	// 3. Liberar PCB
	// Asumiendo que mantenés un map[PID]PCB
	delete(pcbTable, proceso.ID)

	// 4. Intentar inicializar procesos esperando
	h.planilargoplazofifo()
}
