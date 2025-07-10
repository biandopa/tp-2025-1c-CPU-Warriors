package internal

import (
	"log/slog"
	"sync"

	"github.com/sisoputnfrba/tp-golang/cpu/pkg/kernel"
	"github.com/sisoputnfrba/tp-golang/cpu/pkg/memoria"
)

type Service struct {
	Log            *slog.Logger
	Kernel         *kernel.Kernel
	Interrupciones []Interrupcion
	InterruptMutex *sync.RWMutex
	MMU            *MMU
}

func NewService(logger *slog.Logger, ipKernel string, puertoKernel, tlbEntries, cacheEntries int,
	tlbAlgorithm, cacheAlgorithm string, memoriaClient *memoria.Memoria) *Service {
	return &Service{
		Log:            logger,
		Kernel:         kernel.NewKernel(ipKernel, puertoKernel, logger),
		Interrupciones: make([]Interrupcion, 0),
		InterruptMutex: &sync.RWMutex{},
		MMU: NewMMU(tlbEntries, cacheEntries, tlbAlgorithm,
			cacheAlgorithm, logger, memoriaClient),
	}
}
