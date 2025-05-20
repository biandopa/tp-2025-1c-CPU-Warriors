package kernel

import (
	"bytes"
	"context"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/sisoputnfrba/tp-golang/utils/log"
)

type Kernel struct {
	IP     string
	Puerto int
	Log    *slog.Logger
}

func NewKernel(ip string, puerto int, logger *slog.Logger) *Kernel {
	return &Kernel{
		IP:     ip,
		Puerto: puerto,
		Log:    logger,
	}
}

func (k *Kernel) EnviarSyscall(ctx context.Context, body []byte) error {
	url := fmt.Sprintf("http://%s:%d/cpu/proceso", k.IP, k.Puerto)
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(body))
	if err != nil {
		k.Log.ErrorContext(ctx, "Error enviando proceso al Kernel",
			log.StringAttr("ip", k.IP),
			log.IntAttr("puerto", k.Puerto),
			log.ErrAttr(err),
		)
		return err
	}

	if resp != nil {
		k.Log.Debug("Respuesta del servidor recibida.",
			log.StringAttr("status", resp.Status),
			log.AnyAttr("body", string(body)),
		)
	}

	return nil
}
