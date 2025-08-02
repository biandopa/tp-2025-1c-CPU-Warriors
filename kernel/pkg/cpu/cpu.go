package cpu

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/sisoputnfrba/tp-golang/utils/log"
)

type Cpu struct {
	IP         string
	Puerto     int
	ID         string
	Estado     bool
	Log        *slog.Logger
	Proceso    *ProcesoCpu
	httpClient *http.Client
}

type ProcesoCpu struct {
	PID    int    `json:"pid"`
	PC     int    `json:"pc"`
	Motivo string `json:"motivo,omitempty"`
	Rafaga int64  `json:"rafaga,omitempty"`
}

type Interrupcion struct {
	PID            int    `json:"pid"`
	Tipo           string `json:"tipo"`
	EsEnmascarable bool   `json:"es_enmascarable"`
}

func NewCpu(ip string, puerto int, id string, logger *slog.Logger) *Cpu {
	httpClient := &http.Client{
		Timeout: 2 * time.Minute,
	}
	return &Cpu{
		IP:     ip,
		Puerto: puerto,
		ID:     id,
		Estado: true,
		Log:    logger,
		Proceso: &ProcesoCpu{
			PID: -1, // Inicialmente no hay proceso asignado
		},
		httpClient: httpClient,
	}
}

func (c *Cpu) DispatchProcess() (int, string, int64) {
	body, err := json.Marshal(*c.Proceso)
	if err != nil {
		c.Log.Error("Error al serializar el proceso",
			log.ErrAttr(err),
		)
		return c.Proceso.PC, "", 0
	}

	url := fmt.Sprintf("http://%s:%d/kernel/procesos", c.IP, c.Puerto)
	resp, err := c.httpClient.Post(url, "application/json", bytes.NewBuffer(body))
	if err != nil {
		c.Log.Error("error enviando mensaje",
			log.ErrAttr(err),
			log.StringAttr("ip", c.IP),
			log.IntAttr("puerto", c.Puerto),
		)
	}

	newResponse := &ProcesoCpu{}
	if resp != nil {
		c.Log.Debug("Respuesta del servidor",
			log.StringAttr("status", resp.Status),
			log.StringAttr("body", string(body)),
		)

		_ = json.NewDecoder(resp.Body).Decode(newResponse)
		c.Proceso.PID = newResponse.PID
		c.Proceso.PC = newResponse.PC
	}

	return c.Proceso.PC, newResponse.Motivo, newResponse.Rafaga
}

func (c *Cpu) EnviarInterrupcion(tipo string, esEnmascarable bool) bool {
	// Creo una interrupción
	interrupcion := Interrupcion{
		PID:            c.Proceso.PID,
		Tipo:           tipo,
		EsEnmascarable: esEnmascarable,
	}

	// Convierto la estructura del proceso a un []bytes (formato en el que se envían las peticiones)
	body, err := json.Marshal(interrupcion)
	if err != nil {
		c.Log.Error("error codificando mensaje",
			log.ErrAttr(err),
		)
	}

	url := fmt.Sprintf("http://%s:%d/kernel/interrupciones", c.IP, c.Puerto)
	resp, err := c.httpClient.Post(url, "application/json", bytes.NewBuffer(body))
	if err != nil {
		c.Log.Error("error enviando interrupción",
			log.ErrAttr(err),
			log.StringAttr("ip", c.IP),
			log.IntAttr("puerto", c.Puerto),
		)

		return false
	}

	if resp != nil {
		c.Log.Debug("Respuesta del cpu",
			log.StringAttr("status", resp.Status),
			log.StringAttr("body", string(body)),
		)

		if resp.StatusCode != http.StatusOK {
			c.Log.Error("Error al enviar interrupción",
				log.IntAttr("status_code", resp.StatusCode),
			)
			return false
		}

		c.Log.Debug("Interrupción enviada correctamente",
			log.StringAttr("tipo", interrupcion.Tipo),
			log.AnyAttr("es_enmascarable", interrupcion.EsEnmascarable),
		)
	}
	return true
}
