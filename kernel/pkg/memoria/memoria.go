package memoria

type Memoria struct {
	IP     string
	Puerto int
}

func NewMemoria(ip string, puerto int) *Memoria {
	return &Memoria{
		IP:     ip,
		Puerto: puerto,
	}
}

func IntentarCargarEnMemoria(proceso interface{}) bool {
	// Implementar la lógica para intentar cargar el proceso en memoria
	// Retornar true si se carga exitosamente, false en caso contrario
	return true
}
