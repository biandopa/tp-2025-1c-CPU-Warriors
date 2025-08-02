package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/sisoputnfrba/tp-golang/kernel/internal"
	"github.com/sisoputnfrba/tp-golang/utils/log"
)

type rtaCPU struct {
	PID         int      `json:"pid"`
	PC          int      `json:"pc"`
	Instruccion string   `json:"instruccion"`
	Args        []string `json:"args,omitempty"`
}

// crearProceso crea un nuevo proceso con las métricas inicializadas correctamente
func (h *Handler) crearProceso(nombreArchivo, tamanioProceso string) *internal.Proceso {
	proceso := &internal.Proceso{
		PCB: &internal.PCB{
			PID:            h.UniqueID.GetUniqueID(),
			PC:             0,
			MetricasTiempo: map[internal.Estado]*internal.EstadoTiempo{},
			MetricasEstado: map[internal.Estado]int{},
			Tamanio:        tamanioProceso,
			NombreArchivo:  nombreArchivo,
		},
		EstimacionRafaga:     float64(h.Config.InitialEstimate),
		UltimaRafagaEstimada: float64(h.Config.InitialEstimate),
		UltimaRafagaReal:     float64(h.Config.InitialEstimate),
	}

	// Inicializar métricas de tiempo para estado NEW
	proceso.PCB.MetricasTiempo[internal.EstadoNew] = &internal.EstadoTiempo{
		TiempoInicio:    time.Now(),
		TiempoAcumulado: 0,
	}

	// Inicializar contador de estado NEW
	proceso.PCB.MetricasEstado[internal.EstadoNew] = 1

	// Inicializar métricas de tiempo para el resto de estados
	for _, estado := range []internal.Estado{
		internal.EstadoReady,
		internal.EstadoExec,
		internal.EstadoBloqueado,
		internal.EstadoSuspReady,
		internal.EstadoSuspBloqueado,
		internal.EstadoExit,
	} {
		proceso.PCB.MetricasTiempo[estado] = &internal.EstadoTiempo{
			TiempoAcumulado: 0,
		}
	}

	return proceso
}

// EjecutarPlanificadores envia un proceso a la Memoria
func (h *Handler) EjecutarPlanificadores(archivoNombre, tamanioProceso string) {
	// Creo un proceso con métricas inicializadas correctamente
	proceso := h.crearProceso(archivoNombre, tamanioProceso)

	go h.Planificador.PlanificadorLargoPlazo()
	go h.Planificador.PlanificadorCortoPlazo()
	go h.Planificador.SuspenderProcesoBloqueado()
	h.Planificador.CanalNuevoProcesoNew <- proceso

	//Log obligatorio: Creación de proceso
	//"## (<PID>) Se crea el proceso - Estado: NEW"
	h.Log.Info(fmt.Sprintf("## (%d) Se crea el proceso - Estado: NEW", proceso.PCB.PID))
}

func (h *Handler) RespuestaProcesoCPU(w http.ResponseWriter, r *http.Request) {
	decoder := json.NewDecoder(r.Body)
	var syscall rtaCPU

	err := decoder.Decode(&syscall)
	if err != nil {
		h.Log.Error("Error al decodificar la RTA del Proceso",
			log.ErrAttr(err),
		)
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte("Error al decodificar la RTA del Proceso"))
	}

	h.Log.Debug("Me llego la RTA del Proceso",
		log.AnyAttr("syscall", syscall),
	)

	switch syscall.Instruccion {
	case "INIT_PROC":
		//Log obligatorio: Syscall recibida
		//"## (<PID>) - Solicitó syscall: <NOMBRE_SYSCALL>"
		h.Log.Info(fmt.Sprintf("## (%d) - Solicitó syscall: %s", syscall.PID, syscall.Instruccion))

		// Verifico que tenga los argumentos necesarios
		if len(syscall.Args) < 2 {
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte("Error: no se recibieron los argumentos necesarios (archivo y tamaño)"))
			return
		}

		// Creo un proceso hijo con métricas inicializadas correctamente
		proceso := h.crearProceso(syscall.Args[0], syscall.Args[1])

		h.Planificador.CanalNuevoProcesoNew <- proceso

		//Log obligatorio: Creación de proceso
		//"## (<PID>) Se crea el proceso - Estado: NEW"
		h.Log.Info(fmt.Sprintf("## (%d) Se crea el proceso - Estado: NEW", proceso.PCB.PID))

	case "IO":
		ioBuscada := syscall.Args[0] // Nombre de la IO que se busca
		existeIO := false

		ioIdentificacionMutex.RLock()
		for _, io := range ioIdentificacion {
			if io.Nombre == ioBuscada {
				existeIO = true
				break
			}
		}
		ioIdentificacionMutex.RUnlock()

		if !existeIO {
			//No existe la IO, se manda a EXIT
			go h.Planificador.FinalizarProcesoEnCualquierCola(syscall.PID)
			return

		} else {
			// Buscar proceso en EXEC
			if proceso := h.Planificador.BuscarProcesoEnCola(syscall.PID, "EXEC"); proceso == nil {
				h.Log.Debug("Proceso no encontrado en EXEC",
					log.IntAttr("pid", syscall.PID),
				)

				return
			}

			//Log obligatorio: Syscall recibida (solo si el proceso está en EXEC)
			//"## (<PID>) - Solicitó syscall: <NOMBRE_SYSCALL>"
			h.Log.Info(fmt.Sprintf("## (%d) - Solicitó syscall: %s", syscall.PID, syscall.Instruccion))

			//Existe y está libre, pasar a blocked y además manda la señal
			timeSleep, err := strconv.Atoi(syscall.Args[1])
			if err != nil {
				h.Log.Error("Error convirtiendo a int",
					log.ErrAttr(err),
				)
				return
			}

			//Log obligatorio: Motivo de Bloqueo
			//"## (<PID>) - Bloqueado por IO: <DISPOSITIVO_IO>"
			h.Log.Info(fmt.Sprintf("## (%d) - Bloqueado por IO: %s", syscall.PID, ioBuscada))

			//Log obligatorio: Cambio de estado
			// "## (<PID>) Pasa del estado <ESTADO_ANTERIOR> al estado <ESTADO_ACTUAL>"
			h.Log.Info(fmt.Sprintf("## (%d) Pasa del estado EXEC al estado BLOCKED", syscall.PID))

			// Bloquear el proceso
			err = h.Planificador.BloquearPorIO(syscall.PID)
			if err != nil {
				h.Log.Debug("Error al bloquear proceso por IO",
					log.ErrAttr(err),
					log.IntAttr("pid", syscall.PID),
				)

				w.WriteHeader(http.StatusInternalServerError)
				_, _ = w.Write([]byte("{\"error\":\"error al bloquear proceso por IO\"}"))

				return
			}

			// Buscar el dispositivo IO y marcarlo como ocupado. Si todas las instancias de IO con el mismo nombre
			// están ocupadas, se agrega a la cola de espera
			var ioInfo *IOIdentificacion

			ioIdentificacionMutex.Lock()
			for i, ioDevice := range ioIdentificacion {
				if ioDevice.Nombre == ioBuscada && ioDevice.Estado {
					ioIdentificacion[i].Estado = false // Ocupado
					ioIdentificacion[i].ProcesoID = syscall.PID
					ioIdentificacion[i].Cola = "blocked"
					ioInfo = &ioIdentificacion[i]

					h.Log.Debug("Dispositivo IO marcado como ocupado",
						log.StringAttr("dispositivo", ioBuscada),
						log.IntAttr("proceso", syscall.PID),
					)
					break
				}
			}
			ioIdentificacionMutex.Unlock()

			if ioInfo != nil {
				// Enviar petición a IO de forma asíncrona
				go h.Planificador.EnviarUsleep(ioInfo.Puerto, ioInfo.IP, syscall.PID, timeSleep)
			} else {
				// Solo agregar a la cola de espera si NO hay dispositivo libre
				ioWaitQueuesMutex.Lock()
				if ioWaitQueues[ioBuscada] == nil {
					ioWaitQueues[ioBuscada] = make([]IOWaitInfo, 0)
				}
				ioWaitQueues[ioBuscada] = append(ioWaitQueues[ioBuscada], IOWaitInfo{
					PID:       syscall.PID,
					TimeSleep: timeSleep,
				})
				ioWaitQueuesMutex.Unlock()

				h.Log.Debug("Proceso agregado a cola de espera IO (dispositivo ocupado)",
					log.StringAttr("dispositivo", ioBuscada),
					log.IntAttr("proceso", syscall.PID),
					log.IntAttr("tiempo", timeSleep),
					log.IntAttr("cola_espera_size", len(ioWaitQueues[ioBuscada])),
				)
			}

			return
		}

	case "DUMP_MEMORY":
		//Log obligatorio: Syscall recibida
		//"## (<PID>) - Solicitó syscall: <NOMBRE_SYSCALL>"
		h.Log.Info(fmt.Sprintf("## (%d) - Solicitó syscall: %s", syscall.PID, syscall.Instruccion))

		/* Se bloquea el proceso. En caso de error, se envía a la cola de Exit. Caso contrario, se pasa a Ready*/
		go h.Planificador.RealizarDumpMemory(syscall.PID)

	case "EXIT":
		//Log obligatorio: Syscall recibida
		//"## (<PID>) - Solicitó syscall: <NOMBRE_SYSCALL>"
		h.Log.Info(fmt.Sprintf("## (%d) - Solicitó syscall: %s", syscall.PID, syscall.Instruccion))

		go h.Planificador.FinalizarProcesoEnCualquierCola(syscall.PID)

	default:
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte("Instrucción no reconocida"))
		return
	}

	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("ok"))
}
