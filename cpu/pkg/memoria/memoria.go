package memoria

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strconv"

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

type DirInfoResponse struct {
	Pagina int `json:"pagina"`
	Frame  int `json:"frame"`
	Offset int `json:"offset"`
}

type PageConfig struct {
	PageSize       int `json:"page_size"`
	Entries        int `json:"entries_per_page"`
	NumberOfLevels int `json:"number_of_levels"`
}

// LecturaEscrituraBody estructura compatible con el módulo de memoria
type LecturaEscrituraBody struct {
	PID            string `json:"pid"`
	Frame          int    `json:"frame"`
	Offset         int    `json:"offset"`
	Tamanio        int    `json:"tamanio"`
	ValorAEscribir string `json:"valor_a_escribir,omitempty"`
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
func (m *Memoria) Write(pid int, direccion string, datos string, pageConfig PageConfig) error {
	// Convertir dirección física a frame y offset
	dirFisicaInt, err := strconv.Atoi(direccion)
	if err != nil {
		return fmt.Errorf("error al convertir dirección física: %w", err)
	}

	frame := dirFisicaInt / pageConfig.PageSize
	offset := dirFisicaInt % pageConfig.PageSize

	// Crear petición compatible con memoria
	peticion := LecturaEscrituraBody{
		PID:            strconv.Itoa(pid),
		Frame:          frame,
		Offset:         offset,
		ValorAEscribir: datos,
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

	url := fmt.Sprintf("http://%s:%d/cpu/escritura", m.IP, m.Puerto)
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

	// El módulo de memoria responde con "OK" para escrituras exitosas
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		m.Log.Error("Error al leer respuesta WRITE",
			log.ErrAttr(err),
		)
		return fmt.Errorf("error al leer respuesta: %w", err)
	}

	if string(respBody) != "OK" {
		m.Log.Error("WRITE falló en memoria",
			log.StringAttr("respuesta", string(respBody)),
		)
		return fmt.Errorf("WRITE falló: %s", string(respBody))
	}

	m.Log.Debug("WRITE exitoso",
		log.IntAttr("pid", pid),
		log.StringAttr("direccion", direccion),
		log.StringAttr("datos", datos),
	)

	return nil
}

// Read envía una petición de lectura a memoria
func (m *Memoria) Read(pid int, direccion string, tamanio int, pageConfig PageConfig) (string, error) {
	// Convertir dirección física a frame y offset
	dirFisicaInt, err := strconv.Atoi(direccion)
	if err != nil {
		return "", fmt.Errorf("error al convertir dirección física: %w", err)
	}

	frame := dirFisicaInt / pageConfig.PageSize
	offset := dirFisicaInt % pageConfig.PageSize

	// Crear petición compatible con memoria
	peticion := LecturaEscrituraBody{
		PID:     strconv.Itoa(pid),
		Frame:   frame,
		Offset:  offset,
		Tamanio: tamanio,
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

	url := fmt.Sprintf("http://%s:%d/cpu/lectura", m.IP, m.Puerto)
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

	// Decodificar respuesta de lectura
	var respuesta struct {
		Contenido string `json:"contenido"`
	}
	if err = json.NewDecoder(resp.Body).Decode(&respuesta); err != nil {
		m.Log.Error("Error al decodificar respuesta read",
			log.ErrAttr(err),
		)
		return "", fmt.Errorf("error al decodificar respuesta: %w", err)
	}

	m.Log.Debug("READ exitoso",
		log.IntAttr("pid", pid),
		log.StringAttr("direccion", direccion),
		log.IntAttr("tamanio", tamanio),
		log.StringAttr("datos_leidos", respuesta.Contenido),
	)

	return respuesta.Contenido, nil
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

func (m *Memoria) BuscarFrame(pagina, pid int) (DirInfoResponse, error) {
	url := fmt.Sprintf("http://%s:%d/cpu/pagina-a-frame?pid=%d&pagina=%d",
		m.IP, m.Puerto, pid, pagina)

	resp, err := http.Get(url)
	if err != nil {
		m.Log.Error("Error al buscar marco por página",
			log.ErrAttr(err),
			log.StringAttr("ip", m.IP),
			log.IntAttr("puerto", m.Puerto),
			log.IntAttr("pid", pid),
		)
		return DirInfoResponse{}, err
	}
	defer func(Body io.ReadCloser) {
		_ = Body.Close()
	}(resp.Body)

	if resp.StatusCode != http.StatusOK {
		m.Log.Error("Memoria respondió con error al buscar marco",
			log.StringAttr("status", resp.Status),
			log.IntAttr("status_code", resp.StatusCode),
		)
		return DirInfoResponse{}, fmt.Errorf("memoria respondió con error: %s", resp.Status)
	}

	var response DirInfoResponse
	if err = json.NewDecoder(resp.Body).Decode(&response); err != nil {
		m.Log.Error("Error al decodificar respuesta de marco",
			log.ErrAttr(err),
		)
		return response, fmt.Errorf("error al decodificar respuesta: %w", err)
	}

	return response, nil
}

func (m *Memoria) ConsultarPageSize() (PageConfig, error) {
	var info PageConfig
	url := fmt.Sprintf("http://%s:%d//cpu/page-size-y-entries", m.IP, m.Puerto)

	resp, err := http.Get(url)
	if err != nil {
		m.Log.Error("Error al consultar espacio disponible",
			log.ErrAttr(err),
			log.StringAttr("ip", m.IP),
			log.IntAttr("puerto", m.Puerto),
		)
		return info, err
	}
	defer func(Body io.ReadCloser) {
		_ = Body.Close()
	}(resp.Body)

	if resp.StatusCode != http.StatusOK {
		m.Log.Error("Memoria respondió con error al consultar espacio",
			log.StringAttr("status", resp.Status),
			log.IntAttr("status_code", resp.StatusCode),
		)
		return info, fmt.Errorf("memoria respondió con error: %s", resp.Status)
	}

	if err = json.NewDecoder(resp.Body).Decode(&info); err != nil {
		m.Log.Error("Error al decodificar tamaño de página",
			log.ErrAttr(err),
		)
		return info, fmt.Errorf("error al decodificar respuesta: %w", err)
	}

	return info, nil
}

func (m *Memoria) GuardarPagsEnMemoria(info map[int]map[string]interface{}) error {
	url := fmt.Sprintf("http://%s:%d//cpu/actualizar-pag-completa", m.IP, m.Puerto)

	body := new(bytes.Buffer)
	if err := json.NewEncoder(body).Encode(info); err != nil {
		m.Log.Error("Error al serializar información para guardar en memoria",
			log.ErrAttr(err),
		)
		return fmt.Errorf("error al serializar información: %w", err)
	}

	resp, err := http.Post(url, "application/json", body)
	if err != nil {
		m.Log.Error("Error al consultar espacio disponible",
			log.ErrAttr(err),
			log.StringAttr("ip", m.IP),
			log.IntAttr("puerto", m.Puerto),
		)
		return err
	}
	defer func(Body io.ReadCloser) {
		_ = Body.Close()
	}(resp.Body)

	if resp.StatusCode != http.StatusOK {
		m.Log.Error("Memoria respondió con error al guardar información",
			log.StringAttr("status", resp.Status),
			log.IntAttr("status_code", resp.StatusCode),
		)
		return fmt.Errorf("memoria respondió con error: %s", resp.Status)
	}

	return nil
}
