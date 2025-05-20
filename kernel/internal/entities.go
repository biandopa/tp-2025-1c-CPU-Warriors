package internal

import (
	"time"
)

const (
	EstadoNew           Estado = "NEW"
	EstadoReady         Estado = "READY"
	EstadoExec          Estado = "EXEC"
	EstadoBloqueado     Estado = "BLOCKED"
	EstadoSuspReady     Estado = "SUSP.READY"
	EstadoSuspBloqueado Estado = "SUSP.BLOCKED"
	EstadoExit          Estado = "EXIT"
)

type Estado string

type EstadoTiempo struct {
	TiempoInicio    time.Time     `json:"tiempo_inicio"`
	TiempoAcumulado time.Duration `json:"tiempo"`
}

type PCB struct {
	PID            int                      `json:"pid"`
	PC             int                      `json:"pc"`
	MetricasEstado map[Estado]int           `json:"metricas_estado"`
	MetricasTiempo map[Estado]*EstadoTiempo `json:"metricas_tiempo"`
}

type Proceso struct {
	PCB *PCB
}
