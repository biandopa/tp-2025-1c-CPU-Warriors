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
func (s *Service) ObtenerInterrupcion() (Interrupcion, bool) {
	s.InterruptMutex.Lock()
	defer s.InterruptMutex.Unlock()

	if len(s.Interrupciones) == 0 {
		return Interrupcion{}, false
	}

	interrupcion := s.Interrupciones[0]
	s.Interrupciones = s.Interrupciones[1:]

	s.Log.Debug("Interrupción procesada",
		log.StringAttr("tipo", string(interrupcion.Tipo)),
		log.IntAttr("pid", interrupcion.PID),
	)

	return interrupcion, true
}

// LimpiarInterrupciones limpia todas las interrupciones pendientes
func (s *Service) LimpiarInterrupciones() {
	s.InterruptMutex.Lock()
	defer s.InterruptMutex.Unlock()

	s.Interrupciones = make([]Interrupcion, 0)
	s.Log.Debug("Interrupciones limpiadas")
}
