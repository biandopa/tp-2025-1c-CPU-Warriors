package internal

import (
	"log/slog"

	"github.com/sisoputnfrba/tp-golang/cpu/pkg/kernel"
)

type Service struct {
	Log    *slog.Logger
	Kernel *kernel.Kernel
}

func NewService(logger *slog.Logger, ipKernel string, puertoKernel int) *Service {
	return &Service{
		Log:    logger,
		Kernel: kernel.NewKernel(ipKernel, puertoKernel, logger),
	}
}
