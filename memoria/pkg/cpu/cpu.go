package cpu

import (
	"log/slog"
)

type Cpu struct {
	IP     string
	Puerto int
	ID     string
	Log    *slog.Logger
}

func NewCpu(ip string, puerto int, id string, logger *slog.Logger) *Cpu {
	return &Cpu{
		IP:     ip,
		Puerto: puerto,
		ID:     id,
		Log:    logger,
	}
}
