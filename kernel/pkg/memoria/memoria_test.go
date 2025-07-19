package memoria

import (
	"fmt"
	"testing"

	"github.com/jarcoal/httpmock"
	"github.com/sisoputnfrba/tp-golang/utils/log"
)

func TestMemoria_ConsultarEspacio(t *testing.T) {
	m := NewMemoria("1234", 5678, log.BuildLogger("debug"))
	httpmock.Activate(t)
	defer httpmock.DeactivateAndReset()

	type args struct {
		filePath string
		size     string
		pid      int
	}
	tests := []struct {
		name    string
		args    args
		expects func(m *Memoria)
		want    bool
	}{
		{
			name: "Hay espacio en memoria",
			args: args{
				filePath: "/tmp/archivo",
				size:     "1024",
				pid:      1,
			},
			expects: func(m *Memoria) {
				httpmock.RegisterResponder(
					"GET",
					fmt.Sprintf("http://%s:%d/kernel/espacio-disponible", m.IP, m.Puerto),
					httpmock.NewStringResponder(
						200,
						`{"mensaje":"Espacio disponible en memoria","tamaño":1024}`,
					),
				)
			},
			want: true,
		},
		{
			name: "No hay espacio en memoria",
			args: args{
				filePath: "/tmp/archivo",
				size:     "1024",
				pid:      1,
			},
			expects: func(m *Memoria) {
				httpmock.RegisterResponder(
					"GET",
					fmt.Sprintf("http://%s:%d/kernel/espacio-disponible", m.IP, m.Puerto),
					httpmock.NewStringResponder(
						400,
						`{"mensaje":"No hay espacio disponible en memoria"}`,
					),
				)
			},
			want: false,
		},
		{
			name: "Error al consultar espacio en memoria",
			args: args{
				filePath: "/tmp/archivo",
				size:     "1024",
				pid:      1,
			},
			expects: func(m *Memoria) {
				httpmock.RegisterResponder(
					"GET",
					fmt.Sprintf("http://%s:%d/kernel/espacio-disponible", m.IP, m.Puerto),
					httpmock.NewErrorResponder(fmt.Errorf("error al consultar espacio en memoria")),
				)
			},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.expects(m)
			if got := m.ConsultarEspacio(tt.args.size, tt.args.pid); got != tt.want {
				t.Errorf("ConsultarEspacio() = %v, want %v", got, tt.want)
			}
		})
	}
}
