package api

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestHandler_ConsultarEspacioDisponible(t *testing.T) {
	ass := assert.New(t)
	h := NewHandler("../../configs/config.json")

	tests := []struct {
		name         string
		wantedStatus int
		wantedBody   string
	}{
		{
			name:         "Hay espacio disponible",
			wantedStatus: http.StatusOK,
			wantedBody:   `{"mensaje":"Espacio disponible en memoria","tamaño":1024}`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a new request
			req, err := http.NewRequest("GET", "/kernel/espacio-disponible", nil)
			if err != nil {
				t.Fatalf("Error creating request: %v", err)
			}

			// Create a ResponseRecorder to record the response
			rr := httptest.NewRecorder()
			// Create a handler function
			handler := http.HandlerFunc(h.ConsultarEspacioDisponible)
			// Serve the HTTP request
			handler.ServeHTTP(rr, req)

			// Check the status code
			ass.Equal(tt.wantedStatus, rr.Code)
			// Check the response body
			ass.JSONEq(tt.wantedBody, rr.Body.String())
		})
	}
}
