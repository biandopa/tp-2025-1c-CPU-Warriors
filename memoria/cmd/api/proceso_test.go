package api

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
)

func TestHandler_FinalizarProceso(t *testing.T) {
	ass := assert.New(t)
	h := NewHandler("../../configs/config.json")

	type args struct {
		pid string
	}
	tests := []struct {
		name         string
		args         args
		wantedStatus int
		wantedBody   string
	}{
		/*{
			name:         "Finalizar proceso exitoso",
			args:         args{pid: "1234"},
			wantedStatus: http.StatusOK,
			wantedBody:   `{"message":"Proceso 1234 finalizado con éxito"}`,
		},*/
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Configurar el router de forma idéntica a la app real
			r := chi.NewRouter()
			r.Post("/kernel/fin-proceso/{pid}", h.FinalizarProceso)

			// Create a new request
			req, err := http.NewRequest("POST", "/kernel/fin-proceso/"+tt.args.pid, nil)
			if err != nil {
				t.Fatalf("Error creating request: %v", err)
			}

			// Create a ResponseRecorder to record the response
			rr := httptest.NewRecorder()
			// Serve the HTTP request
			r.ServeHTTP(rr, req)

			// Check the status code
			ass.Equal(tt.wantedStatus, rr.Code)
			// Check the response body
			ass.JSONEq(tt.wantedBody, rr.Body.String())
		})
	}
}
