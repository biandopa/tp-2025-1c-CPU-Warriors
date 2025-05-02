package planificadores

func (p *Service) AddCpuConectada(cpu *CpuIdentificacion) {
	// Agregar la CPU a la lista de CPU conectadas
	p.CPUConectadas = append(p.CPUConectadas, cpu)
}
