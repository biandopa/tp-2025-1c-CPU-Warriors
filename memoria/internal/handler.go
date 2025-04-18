package internal

import (
	"log/slog"

	"github.com/sisoputnfrba/tp-golang/utils/log"
)

type Handler struct {
	Log    *slog.Logger
	Config *Config
}

func NewHandler() *Handler {
	return &Handler{
		Log:    log.BuildLogger(),
		Config: IniciarConfiguracion("config.json"),
	}
}
