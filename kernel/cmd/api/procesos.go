package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/sisoputnfrba/tp-golang/kernel/internal"
	"github.com/sisoputnfrba/tp-golang/kernel/internal/planificadores"
	"github.com/sisoputnfrba/tp-golang/utils/log"
)

// EnviarProceso envia un proceso a la Memoria
func (h *Handler) EnviarProceso(archivoNombre, tamanioProceso, args string) {
	// Creo un proceso
	//proceso := internal.Proceso{}

	// TODO: Hacer un switch para elegir un planificador y que ejecute interfaces
	if h.Config.SchedulerAlgorithm == "FIFO" {
		planificador := planificadores.NewPlanificador(h.Log)

		planificador.PlanificadorLargoPlazoFIFO(args)

		// Se ejecuta algun otro planificador
		//lista de procesos en ready, y necesita la lista de cpus
		h.SeleccionarPlanificadorCortoPlazo()

		//planificador.FinalizarProceso(proceso)
	}
}

//LA IDEA ES QUE LA CONSUMA EL PLANIFICADOR CORTO

func (h *Handler) SeleccionarPlanificadorCortoPlazo() {

	switch h.Config.ReadyIngressAlgorithm {
	case "FIFO":
		h.PlanificadorCortoPlazoFIFO()
	case "SJFSD":

	case "SJFD":

	default:
		h.Log.Warn("Algoritmo no reconocido")
	}

}

func (h *Handler) PlanificadorCortoPlazoFIFO() {

	h.Log.Debug("Entre Al PLannificador")

	//TODO: MANDARLO AL PLANFICADOR Y RECIBIR EL PORCESO Y LA CPU DONDE DEBE EJECUTAR
	//PlanificadorCortoPlazoFIFO
	cpu := CPUIdentificacion{
		IP:     "127.0.0.1",
		Puerto: 8004,
		ID:     "CPU-1",
		ESTADO: true,
	}
	//TODO: RECIBIR EL PROCESO A ENVIAR A CPU
	h.enviarProcesoACPU(cpu)
}

func (h *Handler) enviarProcesoACPU(cpuID CPUIdentificacion, proceso *internal.Proceso) {

	h.Log.Debug("Entre al EnviarProceso")
	data := map[string]interface{}{
		"cpuID": cpuID,
		"pc":    proceso.PCB.PID,
		"pid":   proceso.PCB.ProgramCounter, // Cambiar por el ID real
	}

	body, err := json.Marshal(data)
	if err != nil {
		h.Log.Error("Error al serializar ioIdentificacion",
			slog.Attr{Key: "error", Value: slog.StringValue(err.Error())},
		)
		return
	}

	url := fmt.Sprintf("http://%s:%d/kernel/procesos", cpuID.IP, cpuID.Puerto)
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(body))
	if err != nil {
		h.Log.Error("error enviando mensaje",
			slog.Attr{Key: "error", Value: slog.StringValue(err.Error())},
			slog.Attr{Key: "ip", Value: slog.StringValue(cpuID.IP)},
			slog.Attr{Key: "puerto", Value: slog.IntValue(cpuID.Puerto)},
		)
	}

	if resp != nil {
		h.Log.Info("Respuesta del servidor",
			slog.Attr{Key: "status", Value: slog.StringValue(resp.Status)},
			slog.Attr{Key: "body", Value: slog.StringValue(string(body))},
		)
	} else {
		h.Log.Info("Respuesta del servidor: nil")
	}
}

//ESto devuelve el PID + PC + alguno de estos
//IO 25000
//INIT_PROC proceso1 256
//DUMP_MEMORY
//EXIT

type rtaCPU struct {
	PID         int      `json:"pid"`
	PC          int      `json:"pc"`
	Instruccion string   `json:"instruccion"`
	Args        []string `json:"args,omitempty"`
}

func (h *Handler) RespuestaProcesoCPU(w http.ResponseWriter, r *http.Request) {
	decoder := json.NewDecoder(r.Body)
	var proceso rtaCPU

	err := decoder.Decode(&proceso)
	if err != nil {
		h.Log.Error("Error al decodificar la RTA del Proceso",
			log.ErrAttr(err),
		)
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte("Error al decodificar la RTA del Proceso"))
	}

	h.Log.Debug("Me llego la RTA del Proceso",
		log.AnyAttr("proceso", proceso),
	)

	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("ok"))

	switch proceso.Instruccion {
	case "INIT_PROC":
		// TODO: Implementar lógica INIT_PROC (Aca creo un nuevo proceso y los paso a new)
	case "IO":
		// TODO: Implementar lógica IO
	case "DUMP_MEMORY":
		// TODO: Implementar lógica DUMP_MEMORY
	case "EXIT":
		// TODO: Implementar lógica EXIT (aca busco el PID en Exec y lo paso a Exit)
	default:
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte("Instrucción no reconocida"))
		return
	}
}
