package planificadores

import (
	"github.com/sisoputnfrba/tp-golang/kernel/pkg/cpu"
)

func (p *Service) AddCpuConectada(cpuId *CpuIdentificacion) {
	newCPU := cpu.NewCpu(cpuId.IP, cpuId.Puerto, cpuId.ID, p.Log)
	// Agregar la CPU a la lista de CPU conectadas
	p.CPUConectadas = append(p.CPUConectadas, newCPU)
}
