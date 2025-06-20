package cpu

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
)

type Cpu struct {
	IP      string
	Puerto  int
	ID      string
	Estado  bool
	Log     *slog.Logger
	Proceso *ProcesoCpu
}

type ProcesoCpu struct {
	PID int `json:"pid"`
	PC  int `json:"pc"`
}

func NewCpu(ip string, puerto int, id string, logger *slog.Logger) *Cpu {
	return &Cpu{
		IP:      ip,
		Puerto:  puerto,
		ID:      id,
		Estado:  true,
		Log:     logger,
		Proceso: &ProcesoCpu{},
	}
}

func (c *Cpu) DispatchProcess() int {
	body, err := json.Marshal(*c.Proceso)
	if err != nil {
		c.Log.Error("Error al serializar el proceso",
			slog.Attr{Key: "error", Value: slog.StringValue(err.Error())},
		)
		return c.Proceso.PC
	}

	url := fmt.Sprintf("http://%s:%d/kernel/procesos", c.IP, c.Puerto)
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(body))
	if err != nil {
		c.Log.Error("error enviando mensaje",
			slog.Attr{Key: "error", Value: slog.StringValue(err.Error())},
			slog.Attr{Key: "ip", Value: slog.StringValue(c.IP)},
			slog.Attr{Key: "puerto", Value: slog.IntValue(c.Puerto)},
		)
	}

	newResponse := &ProcesoCpu{}
	if resp != nil {
		c.Log.Debug("Respuesta del servidor",
			slog.Attr{Key: "status", Value: slog.StringValue(resp.Status)},
			slog.Attr{Key: "body", Value: slog.StringValue(string(body))},
		)

		_ = json.NewDecoder(resp.Body).Decode(newResponse)
		c.Proceso.PID = newResponse.PID
		c.Proceso.PC = newResponse.PC
	}

	return c.Proceso.PC
}
