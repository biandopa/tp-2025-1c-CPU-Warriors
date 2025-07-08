package planificadores

import (
	"fmt"
	"strings"
	"time"

	"github.com/sisoputnfrba/tp-golang/kernel/internal"
	"github.com/sisoputnfrba/tp-golang/utils/log"
)

func (p *Service) SuspenderProcesoBloqueado() {
	for {
		//La funcion espera que entre un proceso en la cola de blocked
		proceso := <-p.CanalNuevoProcBlocked

		go func() {
			//Busco tiempo de espera para pasar a SuspendedBlocked
			//del archivo de configuración
			tiempoEspera := p.MedianoPlazoConfig.SuspensionTime
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

				//Log obligatorio: Cambio de estado
				// "## (<PID>) Pasa del estado <ESTADO_ANTERIOR> al estado <ESTADO_ACTUAL>"
				p.Log.Info(fmt.Sprintf("## (%d) Pasa del estado BLOCKED al estado SUSP.BLOCKED", proceso.PCB.PID))

				//TODO: Notificar a memoria que debe swappear
				go avisarAMemoriaSwap(proceso)

				//Intentar traer procesos desde SUSP READY o NEW a memoria
				p.CheckearEspacioEnMemoria()

			}
		}()
	}
}

// ManejarFinIO Cuando un proceso en SUSP.BLOCKED finalizo su IO, deberá pasar a
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

		//Log obligatorio: Cambio de estado
		// "## (<PID>) Pasa del estado <ESTADO_ANTERIOR> al estado <ESTADO_ACTUAL>"
		p.Log.Info(fmt.Sprintf("## (%d) Pasa del estado SUSP.BLOCKED al estado SUSP.READY", proceso.PCB.PID))

	} else {
		//Proceso estaba en BLOCKED → READY
		//Esto no se si tiene que estar aca, puede ser logica repetida
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

		//Log obligatorio: Cambio de estado
		// "## (<PID>) Pasa del estado <ESTADO_ANTERIOR> al estado <ESTADO_ACTUAL>"
		p.Log.Info(fmt.Sprintf("## (%d) Pasa del estado BLOCKED al estado READY", proceso.PCB.PID))

		// Notificar planificador corto plazo
		p.canalNuevoProcesoReady <- struct{}{}
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
	// Implementar comunicación con memoria para swapping
	// Por ahora es un placeholder - se completará cuando tengamos el API de memoria
	fmt.Printf("TODO: Notificar a memoria para swappear proceso PID: %d\n", p.PCB.PID)
	// Aquí se debería hacer:
	// 1. Llamar a la API de memoria para mover el proceso a SWAP
	// 2. Actualizar las estructuras administrativas
	// 3. Manejar posibles errores
}

func (p *Service) BuscarProcesoEnCola(pid int, cola string) *internal.Proceso {
	colaString := strings.ToLower(cola)
	switch colaString {
	case "suspended_blocked":
		p.mutexSuspBlockQueue.Lock()
		for _, proc := range p.Planificador.SuspBlockQueue {
			if proc.PCB.PID == pid {
				p.mutexSuspBlockQueue.Unlock()
				return proc
			}
		}
		p.mutexSuspBlockQueue.Unlock()
	case "blocked":
		p.mutexBlockQueue.Lock()
		for _, proc := range p.Planificador.BlockQueue {
			if proc.PCB.PID == pid {
				p.mutexBlockQueue.Unlock()
				return proc
			}
		}
		p.mutexBlockQueue.Unlock()
	case "ready":
		p.mutexReadyQueue.Lock()
		for _, proc := range p.Planificador.ReadyQueue {
			if proc.PCB.PID == pid {
				p.mutexReadyQueue.Unlock()
				return proc
			}
		}
		p.mutexReadyQueue.Unlock()
	case "suspended_ready":
		p.mutexSuspReadyQueue.Lock()
		for _, proc := range p.Planificador.SuspReadyQueue {
			if proc.PCB.PID == pid {
				p.mutexSuspReadyQueue.Unlock()
				return proc
			}
		}
		p.mutexSuspReadyQueue.Unlock()
	default:
		// Si no se especifica cola o es desconocida, buscar en todas las colas relevantes
		// Primero en BLOCKED (más probable para procesos de IO)
		p.mutexBlockQueue.Lock()
		for _, proc := range p.Planificador.BlockQueue {
			if proc.PCB.PID == pid {
				p.mutexBlockQueue.Unlock()
				return proc
			}
		}
		p.mutexBlockQueue.Unlock()

		// Luego en SUSP.BLOCKED
		p.mutexSuspBlockQueue.Lock()
		for _, proc := range p.Planificador.SuspBlockQueue {
			if proc.PCB.PID == pid {
				p.mutexSuspBlockQueue.Unlock()
				return proc
			}
		}
		p.mutexSuspBlockQueue.Unlock()
	}

	p.Log.Error("Proceso no encontrado en la cola especificada",
		log.IntAttr("PID", pid),
		log.StringAttr("Cola", cola),
	)
	return nil
}
