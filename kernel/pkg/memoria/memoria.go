package memoria

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/sisoputnfrba/tp-golang/utils/log"
)

type Memoria struct {
	IP     string
	Puerto int
	Log    *slog.Logger
}

func NewMemoria(ip string, puerto int, logger *slog.Logger) *Memoria {
	return &Memoria{
		IP:     ip,
		Puerto: puerto,
		Log:    logger,
	}
}

func (m *Memoria) ConsultarEspacio(sizeProceso string, pid int) bool {
	url := fmt.Sprintf("http://%s:%d/kernel/espacio-disponible", m.IP, m.Puerto)
	url = fmt.Sprintf("%s?tamanio-proceso=%s&pid=%d", url, sizeProceso, pid)

	resp, err := http.Get(url)
	if err != nil {
		m.Log.Error("Error al consultar espacio en memoria",
			log.ErrAttr(err),
			log.StringAttr("ip", m.IP),
			log.IntAttr("puerto", m.Puerto),
		)
		return false
	}

	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK {
		m.Log.Error("Memoria sin espacio disponible",
			log.StringAttr("ip", m.IP),
			log.IntAttr("puerto", m.Puerto),
			log.IntAttr("status_code", resp.StatusCode),
		)
		return false
	}

	m.Log.Debug("Consulta de espacio en memoria exitosa",
		log.IntAttr("status_code", resp.StatusCode),
	)

	return true
}

func (m *Memoria) CargarProcesoEnMemoriaDeSistema(file string, pid int) bool {
	url := fmt.Sprintf("http://%s:%d/kernel/cargar-memoria-de-sistema", m.IP, m.Puerto)
	url = fmt.Sprintf("%s?archivo=%s&pid=%d", url, file, pid)

	_, err := http.Get(url)
	if err != nil {
		m.Log.Error("Error cargar proceso en memoria de sistema",
			log.ErrAttr(err),
			log.StringAttr("ip", m.IP),
			log.IntAttr("puerto", m.Puerto),
		)
		return false
	}

	return true
}

func (m *Memoria) FinalizarProceso(pid int) (int, error) {
	var (
		status int
		err    error
		resp   *http.Response
	)
	url := fmt.Sprintf("http://%s:%d/kernel/fin-proceso", m.IP, m.Puerto)

	body, _ := json.Marshal(map[string]int{"pid": pid})
	resp, err = http.Post(url, "application/json", bytes.NewBuffer(body))

	if resp != nil {
		status = resp.StatusCode
	}

	return status, err
}

// DumpProceso solicita a memoria realizar un dump del proceso especificado
func (m *Memoria) DumpProceso(pid int) error {
	url := fmt.Sprintf("http://%s:%d/kernel/dump-proceso", m.IP, m.Puerto)
	url = fmt.Sprintf("%s?pid=%d", url, pid)

	resp, err := http.Get(url)
	if err != nil {
		m.Log.Error("Error al solicitar dump de proceso a memoria",
			log.ErrAttr(err),
			log.StringAttr("ip", m.IP),
			log.IntAttr("puerto", m.Puerto),
			log.IntAttr("pid", pid),
		)
		return err
	}

	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK {
		m.Log.Error("Error en dump de proceso - memoria respondió con error",
			log.StringAttr("ip", m.IP),
			log.IntAttr("puerto", m.Puerto),
			log.IntAttr("pid", pid),
			log.IntAttr("status_code", resp.StatusCode),
		)
		return fmt.Errorf("memoria respondió con status %d", resp.StatusCode)
	}

	m.Log.Debug("Dump de proceso realizado exitosamente",
		log.IntAttr("pid", pid),
		log.IntAttr("status_code", resp.StatusCode),
	)

	return nil
}

// SwapProceso solicita a memoria realizar el swap del proceso especificado
func (m *Memoria) SwapProceso(pid int) error {
	url := fmt.Sprintf("http://%s:%d/kernel/swap-proceso?pid=%d", m.IP, m.Puerto, pid)

	resp, err := http.Get(url)

	if err != nil {
		m.Log.Error("Error al solicitar swap de proceso a memoria",
			log.ErrAttr(err),
			log.StringAttr("ip", m.IP),
			log.IntAttr("puerto", m.Puerto),
			log.IntAttr("pid", pid),
		)
		return err
	}

	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK {
		m.Log.Error("Error en swap de proceso - memoria respondió con error",
			log.StringAttr("ip", m.IP),
			log.IntAttr("puerto", m.Puerto),
			log.IntAttr("pid", pid),
			log.IntAttr("status_code", resp.StatusCode),
		)
		return fmt.Errorf("memoria respondió con status %d", resp.StatusCode)
	}

	m.Log.Debug("Swap de proceso realizado exitosamente",
		log.IntAttr("pid", pid),
		log.IntAttr("status_code", resp.StatusCode),
	)

	return nil
}
