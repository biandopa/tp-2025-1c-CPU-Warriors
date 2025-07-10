package api

import "sync"

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

// Estructura para almacenar información de procesos en espera de IO
type IOWaitInfo struct {
	PID       int `json:"pid"`
	TimeSleep int `json:"time_sleep"`
}

// Cola de espera para cada dispositivo IO
var ioWaitQueues map[string][]IOWaitInfo // Mapea nombre de dispositivo a lista de procesos esperando

// Mutex para proteger el acceso concurrente a ioWaitQueues
var ioWaitQueuesMutex sync.RWMutex

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
// Nota: La función init() se ejecuta automáticamente cuando se importa el paquete,
// antes de que se ejecute cualquier otra función
func init() {
	ioWaitQueues = make(map[string][]IOWaitInfo)
}
