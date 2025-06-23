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

	m.Log.Info("Consulta de espacio en memoria exitosa",
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
