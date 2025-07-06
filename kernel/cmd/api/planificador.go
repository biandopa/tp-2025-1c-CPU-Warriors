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
	go h.Planificador.SuspenderProcesoBloqueado() // TODO: ver la parte de
	h.Planificador.CanalNuevoProcesoNew <- proceso

	//Log obligatorio: Creación de proceso
	//“## (<PID>) Se crea el proceso - Estado: NEW”
	h.Log.Info(fmt.Sprintf("%d Se crea el proceso - Estado: NEW", proceso.PCB.PID))
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
	//“## (<PID>) - Solicitó syscall: <NOMBRE_SYSCALL>”
	h.Log.Info(fmt.Sprintf("%d - Solicitó syscall: %s", syscall.PID, syscall.Instruccion))

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
		//“## (<PID>) Se crea el proceso - Estado: NEW”
		h.Log.Info(fmt.Sprintf("%d Se crea el proceso - Estado: NEW", syscall.PID))

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

		} else if ioInfo.Estado {
			//Existe y esta libre, pasar a blocked y ademas manda la señal
			timeSleep, err := strconv.Atoi(syscall.Args[1])
			if err != nil {
				h.Log.Error("Error convirtiendo a int",
					log.ErrAttr(err),
				)
				return
			}
			go h.EnviarPeticionAIO(timeSleep, ioInfo, syscall.PID)

			//TODO proxima entrega: mandar a block

			//Log obligatorio: Motivo de Bloqueo
			//“## (<PID>) - Bloqueado por IO: <DISPOSITIVO_IO>”
			h.Log.Info(fmt.Sprintf("%d - Bloqueado por IO: %s", syscall.PID, ioInfo.Nombre))

			//Log obligatorio: Cambio de estado
			// “## (<PID>) Pasa del estado <ESTADO_ANTERIOR> al estado <ESTADO_ACTUAL>”
			h.Log.Info(fmt.Sprintf("%d Pasa del estado READY al estado BLOCKED", syscall.PID)) //podemos asumir que viene de READY?

			return

		} else {

			//TODO proxima entrega: mandar a block
			fmt.Println("existe y no esta libre")

			//Log obligatorio: Motivo de Bloqueo
			//“## (<PID>) - Bloqueado por IO: <DISPOSITIVO_IO>”
			h.Log.Info(fmt.Sprintf("%d - Bloqueado por IO: %s", syscall.PID, ioInfo.Nombre))

			//Log obligatorio: Cambio de estado
			// “## (<PID>) Pasa del estado <ESTADO_ANTERIOR> al estado <ESTADO_ACTUAL>”
			h.Log.Info(fmt.Sprintf("%d Pasa del estado READY al estado BLOCKED", syscall.PID)) //podemos asumir que viene de READY?

			return
		}

		//
		/* Primero verifica que existe el IO. Si no existe, se manda a EXIT.
		Si existe y está ocupado, se manda a Blocked. Veremos...*/

	case "DUMP_MEMORY":
		// TODO: Implementar lógica DUMP_MEMORY
		// Nota: Este todavía no!!!!!
		/* Esta bloquea el proceso. En caso de error se envía a exit y sino se pasa a Ready*/

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
