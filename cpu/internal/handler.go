package internal

import (
	"log/slog"

	"github.com/sisoputnfrba/tp-golang/utils/log"
)

type Handler struct {
	Log *slog.Logger
}

func NewHandler() *Handler {
	return &Handler{
		Log: log.BuildLogger(),
	}
}
