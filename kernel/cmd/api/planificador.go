package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"sync"

	"github.com/sisoputnfrba/tp-golang/kernel/internal"
	"github.com/sisoputnfrba/tp-golang/utils/log"
)

type rtaCPU struct {
	PID         int      `json:"pid"`
	PC          int      `json:"pc"`
	Instruccion string   `json:"instruccion"`
	Args        []string `json:"args,omitempty"`
}

// EjecutarPlanificadores envia un proceso a la Memoria
func (h *Handler) EjecutarPlanificadores(archivoNombre, tamanioProceso string) {
	// Creo un proceso
	proceso := &internal.Proceso{
		PCB: &internal.PCB{
			PID:                h.UniqueID.GetUniqueID(),
			PC:                 0,
			MetricasTiempo:     map[internal.Estado]*internal.EstadoTiempo{},
			MetricasEstado:     map[internal.Estado]int{},
			Tamanio:            tamanioProceso,
			NombreArchivo:      archivoNombre,
			EstimacionAnterior: float64(h.Config.InitialEstimate),
		},
	}

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

	//Log obligatorio: Syscall recibida
	//"## (<PID>) - Solicitó syscall: <NOMBRE_SYSCALL>"
	h.Log.Info(fmt.Sprintf("## (%d) - Solicitó syscall: %s", syscall.PID, syscall.Instruccion))

	switch syscall.Instruccion {
	case "INIT_PROC":

		// Verifico que tenga los argumentos necesarios
		mu := sync.Mutex{}
		if len(syscall.Args) < 2 {
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte("Error: no se recibieron los argumentos necesarios (archivo y tamaño)"))
			return
		}

		// Creo un proceso hijo
		proceso := &internal.Proceso{
			PCB: &internal.PCB{
				PID:                h.UniqueID.GetUniqueID(),
				PC:                 0,
				MetricasTiempo:     map[internal.Estado]*internal.EstadoTiempo{},
				MetricasEstado:     map[internal.Estado]int{},
				Tamanio:            syscall.Args[1],
				NombreArchivo:      syscall.Args[0],
				EstimacionAnterior: float64(h.Config.InitialEstimate),
			},
		}

		mu.Lock()
		h.Planificador.CanalNuevoProcesoNew <- proceso
		mu.Unlock()

		//Log obligatorio: Creación de proceso
		//"## (<PID>) Se crea el proceso - Estado: NEW"
		h.Log.Info(fmt.Sprintf("## (%d) Se crea el proceso - Estado: NEW", proceso.PCB.PID))

	case "IO":
		var ioInfo IOIdentificacion
		ioBuscada := syscall.Args[0] // Nombre de la IO que se busca
		existeIO := false

		for _, io := range ioIdentificacion {
			if io.Nombre == ioBuscada {
				existeIO = true
				ioInfo = io
				break
			}
		}

		if !existeIO {
			//No existe la IO, se manda a EXIT
			go h.Planificador.FinalizarProceso(syscall.PID)
			return

		} else {
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
			h.Log.Info(fmt.Sprintf("## (%d) - Bloqueado por IO: %s", syscall.PID, ioInfo.Nombre))

			//Log obligatorio: Cambio de estado
			// "## (<PID>) Pasa del estado <ESTADO_ANTERIOR> al estado <ESTADO_ACTUAL>"
			h.Log.Info(fmt.Sprintf("## (%d) Pasa del estado EXEC al estado BLOCKED", syscall.PID))

			// Bloquear el proceso
			err = h.Planificador.BloquearPorIO(syscall.PID)
			if err != nil {
				h.Log.Error("Error al bloquear proceso por IO",
					log.ErrAttr(err),
					log.IntAttr("pid", syscall.PID),
				)

				w.WriteHeader(http.StatusInternalServerError)
				_, _ = w.Write([]byte("{\"error\":\"error al bloquear proceso por IO\"}"))

				return
			}

			// Buscar el dispositivo IO y marcarlo como ocupado. Si todas las instancias de IO con el mismo nomrbe
			// están ocupadas, se agrega a la cola de espera
			var encontreIoLibre bool
			for i, ioDevice := range ioIdentificacion {
				if ioDevice.Nombre == ioInfo.Nombre && ioDevice.Estado == true {
					encontreIoLibre = true
					ioIdentificacion[i].Estado = false // Ocupado
					ioIdentificacion[i].ProcesoID = syscall.PID
					ioIdentificacion[i].Cola = "blocked"

					h.Log.Debug("Dispositivo IO marcado como ocupado",
						log.StringAttr("dispositivo", ioInfo.Nombre),
						log.IntAttr("proceso", syscall.PID),
					)
					break
				}
			}

			// Agregar a la cola de espera del dispositivo IO
			if ioWaitQueues[ioInfo.Nombre] == nil {
				ioWaitQueues[ioInfo.Nombre] = make([]int, 0)
			}
			ioWaitQueues[ioInfo.Nombre] = append(ioWaitQueues[ioInfo.Nombre], syscall.PID)

			h.Log.Debug("Proceso agregado a cola de espera IO",
				log.StringAttr("dispositivo", ioInfo.Nombre),
				log.IntAttr("proceso", syscall.PID),
				log.AnyAttr("cola_espera", ioWaitQueues[ioInfo.Nombre]),
			)

			if encontreIoLibre {
				// Enviar petición a IO de forma asíncrona
				go h.Planificador.EnviarUsleep(ioInfo.Puerto, ioInfo.IP, syscall.PID, timeSleep)
			}

			return
		}

	case "DUMP_MEMORY":
		/* Se bloquea el proceso. En caso de error, se envía a la cola de Exit. Caso contrario, se pasa a Ready*/
		go h.Planificador.RealizarDumpMemory(syscall.PID)

	case "EXIT":
		go h.Planificador.FinalizarProceso(syscall.PID)

	default:
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte("Instrucción no reconocida"))
		return
	}

	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("ok"))
}
