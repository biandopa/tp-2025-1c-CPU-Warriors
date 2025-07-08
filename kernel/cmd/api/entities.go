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
	InitialEstimate       int     `json:"initial_estimate"`
	SuspensionTime        int     `json:"suspension_time"`
	LogLevel              string  `json:"log_level"`
}

// Se usa para almacenar las IOs
var ioIdentificacion []IOIdentificacion

// Cola de espera para cada dispositivo IO
var ioWaitQueues map[string][]int // Mapea nombre de dispositivo a lista de PIDs esperando

// IOIdentificacion EStructura que definimos para manejar las IOs
type IOIdentificacion struct {
	Nombre    string `json:"nombre"`
	IP        string `json:"ip"`
	Puerto    int    `json:"puerto"`
	Estado    bool   `json:"estado"`
	ProcesoID int    `json:"pid"`  // PID del proceso que está usando la IO
	Cola      string `json:"cola"` // Cola a la que pertenece la el proceso (por ejemplo, "ready", "blocked", etc.)
}

// Inicializar las colas de espera para IO
func init() {
	ioWaitQueues = make(map[string][]int)
}
