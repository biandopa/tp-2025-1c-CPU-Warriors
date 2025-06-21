package planificadores

import (
	"time"

	"github.com/sisoputnfrba/tp-golang/kernel/internal"
	"github.com/sisoputnfrba/tp-golang/utils/log"
)

func (p *Service) SuspenderProcesoBloqueado(h *Handler) {
	for {
		//La funcion espera que entre un proceso en la cola de blocked
		proceso := <-p.CanalNuevoProcBlocked
		//TODO: Avisarle al canal cuando se bloquea un proceso
		go func() {
			//Busco tiempo de espera para pasar a SuspendedBlocked
			//del archivo de configuración
			tiempoEspera := h.Config.SuspensionTime
			//Espero que pase el tiempo determinado
			time.Sleep(time.Duration(tiempoEspera) * time.Millisecond)

			//Si el proceso sigue bloqueado, lo suspendemos
			p.mutexBlockQueue.Lock()
			sigueBloqueado := estaEnCola(proceso, p.Planificador.BlockQueue)
			p.mutexBlockQueue.Unlock()

			if sigueBloqueado {
				//Mover de blocked a suspended blocked
				p.mutexBlockQueue.Lock()
				quitarDeCola(&p.Planificador.BlockQueue, proceso)
				p.mutexBlockQueue.Unlock()

				p.mutexSuspBlockQueue.Lock()
				p.Planificador.SuspBlockQueue = append(p.Planificador.SuspBlockQueue, proceso)
				p.mutexSuspBlockQueue.Unlock()

				//Actualizar métricas
				tiempo := proceso.PCB.MetricasTiempo[internal.EstadoBloqueado]
				tiempo.TiempoAcumulado += time.Since(tiempo.TiempoInicio)

				if proceso.PCB.MetricasTiempo[internal.EstadoSuspBloqueado] == nil {
					proceso.PCB.MetricasTiempo[internal.EstadoSuspBloqueado] = &internal.EstadoTiempo{}
				}
				proceso.PCB.MetricasTiempo[internal.EstadoSuspBloqueado].TiempoInicio = time.Now()
				proceso.PCB.MetricasEstado[internal.EstadoSuspBloqueado]++

				//Loggear cambio de estado
				p.Log.Info("Proceso movido de BLCOKED a SUSP_BLOCKED",
					log.IntAttr("PID", proceso.PCB.PID))

				//TODO: Notificar a memoria que debe swappear
				go avisarAMemoriaSwap(proceso)

				//Intentar traer procesos desde SUSP READY o NEW a memoria
				p.CheckearEspacioEnMemoria()

			}
		}()
	}
}

// Cuando un proceso en SUSP.BLOCKED finalizo su IO, deberá pasar a
// SUSP.READY y quedar a la espera de su oportunidad de pasar a READY.
func (p *Service) ManejarFinIO(proceso *internal.Proceso) {

	p.mutexSuspBlockQueue.Lock()
	estabaSuspendido := estaEnCola(proceso, p.Planificador.SuspBlockQueue)
	p.mutexSuspBlockQueue.Unlock()
	if estabaSuspendido {
		p.mutexSuspBlockQueue.Lock()
		quitarDeCola(&p.Planificador.SuspBlockQueue, proceso)
		p.mutexSuspBlockQueue.Unlock()
		p.mutexSuspReadyQueue.Lock()
		p.Planificador.SuspReadyQueue = append(p.Planificador.SuspReadyQueue, proceso)
		p.mutexSuspReadyQueue.Unlock()
		//Notifico que hay un nuevo proceso en SuspReady?
		//p.CanalNewProcSuspReady <- proceso

		//Actualizar métricas
		tiempo := proceso.PCB.MetricasTiempo[internal.EstadoSuspBloqueado]
		tiempo.TiempoAcumulado += time.Since(tiempo.TiempoInicio)

		if proceso.PCB.MetricasTiempo[internal.EstadoSuspReady] == nil {
			proceso.PCB.MetricasTiempo[internal.EstadoSuspReady] = &internal.EstadoTiempo{}
		}
		proceso.PCB.MetricasTiempo[internal.EstadoSuspReady].TiempoInicio = time.Now()
		proceso.PCB.MetricasEstado[internal.EstadoSuspReady]++

		//Loggear cambio de estado
		p.Log.Info("Proceso movido de SUSP_BLOCKED a SUSP_READY",
			log.IntAttr("PID", proceso.PCB.PID))

	} else {
		// Proceso estaba en BLOCKED → READY
		//ESto no se si tiene que estar aca, puede ser logica repetida
		p.mutexBlockQueue.Lock()
		quitarDeCola(&p.Planificador.BlockQueue, proceso)
		p.mutexBlockQueue.Unlock()

		p.mutexReadyQueue.Lock()
		p.Planificador.ReadyQueue = append(p.Planificador.ReadyQueue, proceso)
		p.mutexReadyQueue.Unlock()

		if proceso.PCB.MetricasTiempo[internal.EstadoBloqueado] != nil {
			proceso.PCB.MetricasTiempo[internal.EstadoBloqueado].TiempoAcumulado += time.Since(proceso.PCB.MetricasTiempo[internal.EstadoBloqueado].TiempoInicio)
		}

		if proceso.PCB.MetricasTiempo[internal.EstadoReady] == nil {
			proceso.PCB.MetricasTiempo[internal.EstadoReady] = &internal.EstadoTiempo{}
		}
		proceso.PCB.MetricasTiempo[internal.EstadoReady].TiempoInicio = time.Now()
		proceso.PCB.MetricasEstado[internal.EstadoReady]++

		p.Log.Info("Proceso movido de BLOCKED a READY",
			log.IntAttr("PID", proceso.PCB.PID))

		// Notificar planificador corto plazo
		p.canalNuevoProcesoReady <- proceso
	}
}

func estaEnCola(p *internal.Proceso, cola []*internal.Proceso) bool {
	for _, proc := range cola {
		if proc.PCB.PID == p.PCB.PID {
			return true
		}
	}
	return false
}

func quitarDeCola(cola *[]*internal.Proceso, p *internal.Proceso) {
	for i, proc := range *cola {
		if proc.PCB.PID == p.PCB.PID {
			*cola = append((*cola)[:i], (*cola)[i+1:]...)
			return
		}
	}
}

// TODO
func avisarAMemoriaSwap(p *internal.Proceso) {

}
