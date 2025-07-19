package api

type Config struct {
	IpKernel   string `json:"ip_kernel"`
	PortKernel int    `json:"port_kernel"`
	PortIo     int    `json:"port_io"`
	IpIo       string `json:"ip_io"`
	LogLevel   string `json:"log_level"`
}

type IOIdentificacion struct {
	Nombre    string `json:"nombre"`
	IP        string `json:"ip"`
	Puerto    int    `json:"puerto"`
	ProcesoID int    `json:"pid"`  // PID del proceso que está usando la IO
	Cola      string `json:"cola"` // Cola a la que pertenece el proceso
}

type Usleep struct {
	PID         int `json:"pid"`
	TiempoSleep int `json:"tiempo_sleep"`
}
