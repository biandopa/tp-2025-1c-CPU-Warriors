package api

type Config struct {
	IpMemory              string  `json:"ip_memory"`
	PortMemory            int     `json:"port_memory"`
	IpKernel              string  `json:"ip_kernel"`
	PortKernel            int     `json:"port_kernel"`
	IpIo                  string  `json:"ip_io"`
	PortIo                int     `json:"port_io"`
	IpCPU                 string  `json:"ip_cpu"`
	PortCPU               int     `json:"port_cpu"`
	SchedulerAlgorithm    string  `json:"scheduler_algorithm"`
	ReadyIngressAlgorithm string  `json:"ready_ingress_algorithm"`
	Alpha                 float64 `json:"alpha"`
	SuspensionTime        int     `json:"suspension_time"`
	LogLevel              string  `json:"log_level"`
}

// TODO: HACER UNA LISTA DE IO
var ioIdentificacion IOIdentificacion

type IOIdentificacion struct {
	Nombre string `json:"nombre"`
	IP     string `json:"ip"`
	Puerto int    `json:"puerto"`
}

type CPUIdentificacion struct {
	IP     string `json:"ip"`
	Puerto int    `json:"puerto"`
	ID     string `json:"id"`
}
