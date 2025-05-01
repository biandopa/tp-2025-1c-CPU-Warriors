package memoria

import (
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

func (m *Memoria) ConsultarEspacio() bool {
	url := fmt.Sprintf("http://%s:%d/kernel/espacio-disponible", m.IP, m.Puerto)

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
