package internal

import (
	"encoding/json"
)

// EnviarProcesoSyscall envia un proceso al kernel
func (s *Service) EnviarProcesoSyscall(syscall *ProcesoSyscall) error {
	// Conviero la estructura del proceso a un []bytes (formato en el que se envían las peticiones)
	body, _ := json.Marshal(syscall)
	// Envio la syscall al kernel
	return s.Kernel.EnviarSyscall(body)
}
