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
			PID:            0,
			PC:             0,
			MetricasTiempo: map[internal.Estado]*internal.EstadoTiempo{},
			MetricasEstado: map[internal.Estado]int{},
			Tamanio:        tamanioProceso,
			NombreArchivo:  archivoNombre,
		},
	}

	go h.Planificador.PlanificadorLargoPlazo(h.Config.ReadyIngressAlgorithm)
	h.ejecutarPlanificadorCortoPlazo()
	h.Planificador.CanalNuevoProcesoNew <- proceso
}

// ejecutarPlanificadorCortoPlazo selecciona el planificador de corto plazo a utilizar y lo ejecuta como una goroutine.
func (h *Handler) ejecutarPlanificadorCortoPlazo() {
	switch h.Config.ReadyIngressAlgorithm {
	case "FIFO":
		go h.Planificador.PlanificadorCortoPlazoFIFO()
	case "SJFSD":

	case "SJFD":
	case "PMCP":

	default:
		h.Log.Warn("Algoritmo no reconocido")
	}
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
		mu := sync.Mutex{}
		if len(syscall.Args) < 2 {
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte("Error: no se recibieron los argumentos necesarios (archivo y tamaño)"))
			return
		}

		// Creo un proceso hijo
		proceso := &internal.Proceso{
			PCB: &internal.PCB{
				PID:            1,
				PC:             0,
				MetricasTiempo: map[internal.Estado]*internal.EstadoTiempo{},
				MetricasEstado: map[internal.Estado]int{},
			},
		}

		mu.Lock()
		h.Planificador.CanalNuevoProcesoNew <- proceso
		mu.Unlock()
	case "IO":
		var ioInfo IOIdentificacion
		ioBuscada := syscall.Args[0]
		existeIO := false

		for _, io := range ioIdentificacion {
			if io.Nombre == ioBuscada {
				existeIO = true
				ioInfo = io
				break
			}
		}

		if !existeIO {
			go h.Planificador.FinalizarProceso(syscall.PID)
			return
		} else if ioInfo.Estado {
			//Existe y esta libre, blocked y ademas manda la señal
			timeSleep, err := strconv.Atoi(syscall.Args[1])
			if err != nil {
				h.Log.Error("Error convirtiendo a int",
					log.ErrAttr(err),
				)
				return
			}
			h.EnviarPeticionAIO(timeSleep, ioInfo, syscall.PID)

			//TODO proxima entrega: mandar a block
			return
		} else {
			//TODO proxima entrega: mandar a block
			fmt.Println("existe y no esta libre")
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
