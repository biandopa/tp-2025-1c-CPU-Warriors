package internal

import "github.com/sisoputnfrba/tp-golang/utils/log"

func (s *Service) AgregarInterrupcion(interrupcion Interrupcion) {
	s.InterruptMutex.Lock()
	defer s.InterruptMutex.Unlock()

	s.Interrupciones = append(s.Interrupciones, interrupcion)
	s.Log.Debug("Interrupción agregada",
		log.StringAttr("tipo", string(interrupcion.Tipo)),
		log.IntAttr("pid", interrupcion.PID))
}

func (s *Service) HayInterrupciones() bool {
	s.InterruptMutex.RLock()
	defer s.InterruptMutex.RUnlock()
	return len(s.Interrupciones) > 0
}

// ObtenerInterrupcion obtiene y elimina la primera interrupción de la cola
func (s *Service) ObtenerInterrupcion(pid int) (Interrupcion, bool) {
	s.InterruptMutex.Lock()
	defer s.InterruptMutex.Unlock()

	if len(s.Interrupciones) == 0 {
		return Interrupcion{}, false
	}

	// Si hay interrupciones para el PID específico, devolver la primera y eliminarla
	var (
		interrupcion Interrupcion
		index        int
		found        bool
	)

	for i, interr := range s.Interrupciones {
		if interr.PID == pid {
			s.Log.Debug("Interrupción encontrada para PID",
				log.IntAttr("pid", pid),
				log.StringAttr("tipo", string(interrupcion.Tipo)))

			index = i
			interrupcion = interr
			found = true
			break
		}
	}

	s.Interrupciones = append(s.Interrupciones[:index], s.Interrupciones[index+1:]...)

	s.Log.Debug("Interrupción procesada",
		log.StringAttr("tipo", string(interrupcion.Tipo)),
		log.IntAttr("pid", interrupcion.PID),
	)

	return interrupcion, found
}

// LimpiarInterrupciones limpia todas las interrupciones pendientes
func (s *Service) LimpiarInterrupciones() {
	s.InterruptMutex.Lock()
	defer s.InterruptMutex.Unlock()

	s.Interrupciones = make([]Interrupcion, 0)
	s.Log.Debug("Interrupciones limpiadas")
}
