package internal

const (
	InterrupcionExcepcion TipoDeInterrupcion = "Excepcion"
	InterrupcionExterna   TipoDeInterrupcion = "Externa"
	InerrupcionDesalojo   TipoDeInterrupcion = "Desalojo"
	InterrupcionFinIO     TipoDeInterrupcion = "FinIO"
)

type ProcesoSyscall struct {
	PID         int      `json:"pid"`
	PC          int      `json:"pc"`
	Instruccion string   `json:"instruccion"`
	Args        []string `json:"args,omitempty"`
}

type Interrupcion struct {
	PID            int                `json:"pid"`
	Tipo           TipoDeInterrupcion `json:"tipo"`
	EsEnmascarable bool               `json:"es_enmascarable"`
}

type TipoDeInterrupcion string
