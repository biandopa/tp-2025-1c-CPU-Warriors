package api

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/sisoputnfrba/tp-golang/utils/log"
)

func (h *Handler) FinalizarProceso(w http.ResponseWriter, r *http.Request) {

	var (
		// Leemos el PID
		pid = r.URL.Query().Get("pid")
	)

	if pid == "" {
		h.Log.Error("PID no proporcionado")
		http.Error(w, "PID no proporcionado", http.StatusBadRequest)
		return
	}

	tablaMetricas, err := h.BuscarProcesoPorPID(pid)
	if err != nil {
		h.Log.Error("Error al buscar el proceso por PID",
			log.ErrAttr(err),
			log.StringAttr("pid", pid),
		)
		http.Error(w, "Proceso no encontrado", http.StatusNotFound)
		return
	}
	/* Log obligatorio: Destrucción de Proceso
	“## PID: <PID> - Proceso Destruido - Métricas - Acc.T.Pag: <ATP>;
	Inst.Sol.: <Inst.Sol.>; SWAP: <SWAP>; Mem.Prin.: <Mem.Prin.>; Lec.Mem.: <Lec.Mem.>; Esc.Mem.: <Esc.Mem.>”*/
	h.Log.Info(fmt.Sprintf("## PID: %s - Proceso Destruido - Métricas - Acc.T.Pag: %d; "+
		"Inst.Sol.: %d; SWAP: %d; Mem.Prin.: %d; Lec.Mem.: %d; Esc.Mem.: %d",
		pid, tablaMetricas.CantidadAccesosATablas, tablaMetricas.CantidadInstruccionesSolicitadas,
		tablaMetricas.CantidadBajadasSwap, tablaMetricas.CantidadSubidasMemoriaPrincipal,
		tablaMetricas.CantidadDeLectura, tablaMetricas.CantidadDeEscritura))

	h.finalizarProcesoFuncionAuxiliar(pid)

	// Enviamos una respuesta exitosa
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("Proceso finalizado correctamente"))
}

func (h *Handler) finalizarProcesoFuncionAuxiliar(pid string) {
	pidInt, _ := strconv.Atoi(pid)

	if h.ContienePIDEnSwap(pidInt) {
		//compactar la posicion en swap y borrarlo en la lista de procesos
		if err := h.CompactarSwap(pid); err != nil {
			h.Log.Error("Error al compactar swap", log.ErrAttr(err))
		}

	} else {
		procesYTablaAsociada, _ := h.BuscarProcesoPorPID(pid)
		h.Log.Debug("FinalizarProcesoFuncionAuxiliar",
			log.AnyAttr("procesYTablaAsociada", procesYTablaAsociada.TablasDePaginas))

		marcosDelProceso := h.ObtenerMarcosDeLaTabla(procesYTablaAsociada.TablasDePaginas)

		for _, marco := range marcosDelProceso {
			copy(h.EspacioDeUsuario[marco*h.Config.PageSize:(marco+1)*h.Config.PageSize], make([]byte, h.Config.PageSize))
		}

		// Actualizar el espacio de usuario con los marcos libres
		h.Log.Info("Liberando marcos de memoria",
			log.AnyAttr("marcos_del_proceso", marcosDelProceso),
		)
		for _, marco := range marcosDelProceso {
			h.FrameTable[marco] = false // Marcar el marco como libre
		}
		h.Log.Info("Marcos liberados correctamente",
			log.AnyAttr("frame_table", h.FrameTable),
		)

	}

	//hasta aca el else
	//2do borrarlo de la lista de tablas
	if err := h.borrarProcesoPorPID(pid); err != nil {
		h.Log.Error("Error al borrar el proceso por PID", log.ErrAttr(err))
		return
	}
	//3ero borrar las instrucciones

	delete(h.Instrucciones, pidInt)
}

func (h *Handler) borrarProcesoPorPID(pid string) error {
	for i, proceso := range h.TablasProcesos {
		if proceso.PID == pid {
			// Borramos el elemento del slice
			h.TablasProcesos = append(h.TablasProcesos[:i], h.TablasProcesos[i+1:]...)
			return nil
		}
	}
	return fmt.Errorf("proceso con PID %s no encontrado", pid)
}
