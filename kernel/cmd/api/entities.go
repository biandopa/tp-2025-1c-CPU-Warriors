package api

type IOIdentificacion struct {
	Nombre string `json:"nombre"`
	IP     string `json:"ip"`
	Puerto int    `json:"puerto"`
}

type Config struct {
	IpMemory              string `json:"ip_memory"`
	PortMemory            int    `json:"port_memory"`
	IpKernel              string `json:"ip_kernel"`
	PortKernel            int    `json:"port_kernel"`
	SchedulerAlgorithm    string `json:"scheduler_algorithm"`
	ReadyIngressAlgorithm int    `json:"ready_ingress_algorithm"`
	Alpha                 int    `json:"alpha"`
	SuspensionTime        int    `json:"suspension_time"`
	LogLevel              string `json:"log_level"`
}

// TODO: HACER UNA LISTA DE IO
var ioIdentificacion IOIdentificacion

var identificacionCPU = map[string]interface{}{
	"ip":     "",
	"puerto": "",
	"id":     "",
}
