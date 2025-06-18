package api

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestHandler_ConsultarEspacioDisponible(t *testing.T) {
	ass := assert.New(t)
	h := NewHandler("../../configs/config-test.json")

	type args struct {
		archivo        string
		tamanioProceso string
		pid            string
	}
	tests := []struct {
		name         string
		args         args
		wantedStatus int
		wantedBody   string
	}{
		{
			name:         "Hay espacio disponible",
			args:         args{archivo: "../../examples/proceso1", tamanioProceso: "10", pid: "1"},
			wantedStatus: http.StatusOK,
			wantedBody:   "{\"mensaje\":\"Espacio disponible en memoria\",\"tamaño\":1024}\n",
		},
		{
			name:         "No existe el archivo",
			args:         args{archivo: "../../examples/no-existe", tamanioProceso: "10", pid: "1"},
			wantedStatus: http.StatusInternalServerError,
			wantedBody:   "error al abrir el archivo de pseudocodigo\n",
		},
		{
			name:         "Tamaño del proceso no proporcionado",
			args:         args{archivo: "../../examples/proceso1", tamanioProceso: "", pid: "1"},
			wantedStatus: http.StatusBadRequest,
			wantedBody:   "tamaño del proceso no proporcionado\n",
		},
		{
			name:         "Archivo de pseudocódigo no proporcionado",
			args:         args{archivo: "", tamanioProceso: "10", pid: "1"},
			wantedStatus: http.StatusBadRequest,
			wantedBody:   "archivo de pseudocódigo no proporcionado\n",
		},
		{
			name:         "PID no proporcionado",
			args:         args{archivo: "../../examples/proceso1", tamanioProceso: "10", pid: ""},
			wantedStatus: http.StatusBadRequest,
			wantedBody:   "PID no proporcionado\n",
		},
		{
			name:         "Error al convertir PID a entero",
			args:         args{archivo: "../../examples/proceso1", tamanioProceso: "10", pid: "not-an-int"},
			wantedStatus: http.StatusBadRequest,
			wantedBody:   "error al convertir PID a entero\n",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a new request
			req, err := http.NewRequest("GET",
				fmt.Sprintf("/kernel/espacio-disponible?archivo=%s&tamanio-proceso=%s&pid=%s",
					tt.args.archivo, tt.args.tamanioProceso, tt.args.pid), nil)
			if err != nil {
				t.Fatalf("Error creating request: %v", err)
			}

			// Create a ResponseRecorder to record the response
			rr := httptest.NewRecorder()
			// Create a handler function
			handler := http.HandlerFunc(h.ConsultarEspacioEInicializar)
			// Serve the HTTP request
			handler.ServeHTTP(rr, req)

			// Check the status code
			ass.Equal(tt.wantedStatus, rr.Code)
			// Check the response body
			ass.Equal(tt.wantedBody, rr.Body.String())
		})
	}
}
