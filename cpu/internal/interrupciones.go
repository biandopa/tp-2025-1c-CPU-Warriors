package internal

func (s *Service) AgregarInterrupcion(interrupcion Interrupcion) {
	s.Interrupciones = append(s.Interrupciones, interrupcion)
	s.Log.Debug("Interrupción agregada", "tipo", interrupcion.Tipo, "PID", interrupcion.PID)
}

func (s *Service) HayInterrupciones() bool {
	return len(s.Interrupciones) > 0
}
