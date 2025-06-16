package api

type Config struct {
	PortMemory     int    `json:"port_memory"`
	IpMemory       string `json:"ip_memory"`
	IpCpu          string `json:"ip_cpu"`
	PortCpu        int    `json:"port_cpu"`
	MemorySize     int    `json:"memory_size"`
	PageSize       int    `json:"page_size"`
	EntriesPerPage int    `json:"entries_per_page"`
	NumberOfLevels int    `json:"number_of_levels"`
	MemoryDelay    int    `json:"memory_delay"`
	SwapfilePath   string `json:"swapfile_path"`
	SwapDelay      int    `json:"swap_delay"`
	LogLevel       string `json:"log_level"`
	DumpPath       string `json:"dump_path"`
	ScriptsPath    string `json:"scripts_path"`
}

type MetricasProceso struct {
	AccesosTablaDePaginas    int `json:"accesos_tabla_de_paginas"`
	InstruccionesSolicitadas int `json:"instrucciones_solicitadas"`
	BajadasAlSwap            int `json:"bajadas_a_swap"`
	SubidasAMemPpal          int `json:"subidas_a_memoria_principal"`
	LecturasDeMemoria        int `json:"lecturas_de_memoria"`
	EscriturasDeMemoria      int `json:"escrituras_de_memoria"`
}

type Instruccion struct {
	Instruccion string   `json:"instruccion"`
	Parametros  []string `json:"parametros"`
}
