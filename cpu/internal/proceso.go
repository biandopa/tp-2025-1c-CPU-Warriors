package internal

import (
	"encoding/json"
	"github.com/sisoputnfrba/tp-golang/utils/log"
)

// EnviarProcesoSyscall envia un proceso al kernel
func (s *Service) EnviarProcesoSyscall(syscall *ProcesoSyscall) error {
	// Conviero la estructura del proceso a un []bytes (formato en el que se envían las peticiones)
	body, _ := json.Marshal(syscall)
	// Envio la syscall al kernel
	return s.Kernel.EnviarSyscall(body)
}

// LimpiarMemoriaProceso limpia la memoria (TLB y caché) cuando se desaloja un proceso
func (s *Service) LimpiarMemoriaProceso(pid int) {
	s.Log.Debug("Solicitando limpieza de memoria por desalojo de proceso",
		log.IntAttr("pid", pid))

	s.MMU.LimpiarMemoriaProceso(pid)
}
