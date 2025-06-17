package memoria

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"

	"github.com/sisoputnfrba/tp-golang/utils/log"
)

// Memoria representa el cliente para comunicarse con el módulo de memoria
type Memoria struct {
	IP     string
	Puerto int
	Log    *slog.Logger
}

// PeticionAcceso representa una petición de acceso a memoria
type PeticionAcceso struct {
	PID       int    `json:"pid"`
	Direccion string `json:"direccion"`
	Datos     string `json:"datos,omitempty"`   // Solo para WRITE
	Tamanio   int    `json:"tamanio,omitempty"` // Solo para READ
	Operacion string `json:"operacion"`         // "READ" o "WRITE"
}

// RespuestaAcceso representa la respuesta de memoria
type RespuestaAcceso struct {
	Datos   string `json:"datos,omitempty"` // Solo para READ
	Exito   bool   `json:"exito"`
	Mensaje string `json:"mensaje,omitempty"`
}

// PeticionInstruccion representa una petición de instrucción
type PeticionInstruccion struct {
	PID int `json:"pid"`
	PC  int `json:"pc"`
}

// Instruccion representa una instrucción devuelta por memoria
type Instruccion struct {
	Instruccion string   `json:"instruccion"`
	Parametros  []string `json:"parametros"`
}

// NewMemoria crea una nueva instancia del cliente de memoria
func NewMemoria(ip string, puerto int, logger *slog.Logger) *Memoria {
	return &Memoria{
		IP:     ip,
		Puerto: puerto,
		Log:    logger,
	}
}

// Write envía una petición de escritura a memoria
func (m *Memoria) Write(pid int, direccion string, datos string) error {
	peticion := PeticionAcceso{
		PID:       pid,
		Direccion: direccion,
		Datos:     datos,
		Operacion: "WRITE",
	}

	body, err := json.Marshal(peticion)
	if err != nil {
		m.Log.Error("Error al serializar petición WRITE",
			log.IntAttr("pid", pid),
			log.StringAttr("direccion", direccion),
			log.ErrAttr(err),
		)
		return fmt.Errorf("error al serializar petición WRITE: %w", err)
	}

	url := fmt.Sprintf("http://%s:%d/cpu/acceso", m.IP, m.Puerto)
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(body))
	if err != nil {
		m.Log.Error("Error enviando petición WRITE a memoria",
			log.StringAttr("ip", m.IP),
			log.IntAttr("puerto", m.Puerto),
			log.IntAttr("pid", pid),
			log.StringAttr("direccion", direccion),
			log.ErrAttr(err),
		)
		return fmt.Errorf("error al enviar petición WRITE: %w", err)
	}
	defer func(Body io.ReadCloser) {
		_ = Body.Close()
	}(resp.Body)

	if resp.StatusCode != http.StatusOK {
		m.Log.Error("Memoria respondió con error en WRITE",
			log.StringAttr("status", resp.Status),
			log.IntAttr("status_code", resp.StatusCode),
		)
		return fmt.Errorf("memoria respondió con error: %s", resp.Status)
	}

	var respuesta RespuestaAcceso
	if err = json.NewDecoder(resp.Body).Decode(&respuesta); err != nil {
		m.Log.Error("Error al decodificar respuesta WRITE",
			log.ErrAttr(err),
		)
		return fmt.Errorf("error al decodificar respuesta: %w", err)
	}

	if !respuesta.Exito {
		m.Log.Error("WRITE falló en memoria",
			log.StringAttr("mensaje", respuesta.Mensaje),
		)
		return fmt.Errorf("WRITE falló: %s", respuesta.Mensaje)
	}

	m.Log.Debug("WRITE exitoso",
		log.IntAttr("pid", pid),
		log.StringAttr("direccion", direccion),
		log.StringAttr("datos", datos),
	)

	return nil
}

// Read envía una petición de lectura a memoria
func (m *Memoria) Read(pid int, direccion string, tamanio int) (string, error) {
	peticion := PeticionAcceso{
		PID:       pid,
		Direccion: direccion,
		Tamanio:   tamanio,
		Operacion: "READ",
	}

	body, err := json.Marshal(peticion)
	if err != nil {
		m.Log.Error("Error al serializar petición READ",
			log.IntAttr("pid", pid),
			log.StringAttr("direccion", direccion),
			log.IntAttr("tamanio", tamanio),
			log.ErrAttr(err),
		)
		return "", fmt.Errorf("error al serializar petición READ: %w", err)
	}

	url := fmt.Sprintf("http://%s:%d/cpu/acceso", m.IP, m.Puerto)
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(body))
	if err != nil {
		m.Log.Error("Error enviando petición read a memoria",
			log.StringAttr("ip", m.IP),
			log.IntAttr("puerto", m.Puerto),
			log.IntAttr("pid", pid),
			log.StringAttr("direccion", direccion),
			log.ErrAttr(err),
		)
		return "", fmt.Errorf("error al enviar petición read: %w", err)
	}
	defer func(Body io.ReadCloser) {
		_ = Body.Close()
	}(resp.Body)

	if resp.StatusCode != http.StatusOK {
		m.Log.Error("Memoria respondió con error en read",
			log.StringAttr("status", resp.Status),
			log.IntAttr("status_code", resp.StatusCode),
		)
		return "", fmt.Errorf("memoria respondió con error: %s", resp.Status)
	}

	var respuesta RespuestaAcceso
	if err = json.NewDecoder(resp.Body).Decode(&respuesta); err != nil {
		m.Log.Error("Error al decodificar respuesta read",
			log.ErrAttr(err),
		)
		return "", fmt.Errorf("error al decodificar respuesta: %w", err)
	}

	if !respuesta.Exito {
		m.Log.Error("READ falló en memoria",
			log.StringAttr("mensaje", respuesta.Mensaje),
		)
		return "", fmt.Errorf("READ falló: %s", respuesta.Mensaje)
	}

	m.Log.Debug("READ exitoso",
		log.IntAttr("pid", pid),
		log.StringAttr("direccion", direccion),
		log.IntAttr("tamanio", tamanio),
		log.StringAttr("datos_leidos", respuesta.Datos),
	)

	return respuesta.Datos, nil
}

// FetchInstruccion obtiene una instrucción de memoria
func (m *Memoria) FetchInstruccion(pid int, pc int) (Instruccion, error) {
	var instruccion Instruccion

	peticion := PeticionInstruccion{
		PID: pid,
		PC:  pc,
	}

	body, err := json.Marshal(peticion)
	if err != nil {
		m.Log.Error("Error al serializar petición de instrucción",
			log.IntAttr("pid", pid),
			log.IntAttr("pc", pc),
			log.ErrAttr(err),
		)
		return instruccion, fmt.Errorf("error al serializar petición: %w", err)
	}

	url := fmt.Sprintf("http://%s:%d/cpu/instruccion", m.IP, m.Puerto)
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(body))
	if err != nil {
		m.Log.Error("Error enviando petición de instrucción",
			log.StringAttr("ip", m.IP),
			log.IntAttr("puerto", m.Puerto),
			log.IntAttr("pid", pid),
			log.IntAttr("pc", pc),
			log.ErrAttr(err),
		)
		return instruccion, fmt.Errorf("error al enviar petición: %w", err)
	}
	defer func(Body io.ReadCloser) {
		_ = Body.Close()
	}(resp.Body)

	if resp.StatusCode != http.StatusOK {
		m.Log.Error("Memoria respondió con error en fetch",
			log.StringAttr("status", resp.Status),
			log.IntAttr("status_code", resp.StatusCode),
		)
		return instruccion, fmt.Errorf("memoria respondió con error: %s", resp.Status)
	}

	if err = json.NewDecoder(resp.Body).Decode(&instruccion); err != nil {
		m.Log.Error("Error al decodificar instrucción",
			log.ErrAttr(err),
		)
		return instruccion, fmt.Errorf("error al decodificar instrucción: %w", err)
	}

	m.Log.Debug("FETCH exitoso",
		log.IntAttr("pid", pid),
		log.IntAttr("pc", pc),
		log.StringAttr("instruccion", instruccion.Instruccion),
		log.AnyAttr("parametros", instruccion.Parametros),
	)

	return instruccion, nil
}
