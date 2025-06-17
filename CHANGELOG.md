# 📝 Changelog

## **Fecha:** 2025-06-21

---

### 🚀 **Cambios Principales - Checkpoint 2 + Refactoring**

#### **1. Módulo CPU - Instrucciones READ/WRITE Implementadas**

##### **📁 Archivo:** `cpu/cmd/api/instruciones.go`

**🔧 Cambios realizados:**

1. **Implementación completa de instrucción WRITE:**
   ```go
   case "WRITE":
   	// ❌ ANTES: Código comentado
   	/*direccion := args[0]
   	datos := args[1]
   	dirFisica := traducirDireccion(pid, direccion)
   	h.writeMemoria("pid", pid, dirFisica, datos)
   	//TODO: implementar traducirDireccion, writeMemoria*/
   	
   	// ✅ DESPUÉS: Implementación completa con módulo dedicado
   	if len(args) < 2 {
   		h.Log.Error("WRITE requiere al menos 2 argumentos: dirección y datos")
   		return false, pc
   	}
   	direccion := args[0]
   	datos := args[1]
   	dirFisica := direccion // TODO: implementar traducción
   	
   	if err := h.Memoria.Write(pid, dirFisica, datos); err != nil {
   		// Manejo de errores completo
   		return false, pc
   	}
   	nuevoPC = pc + 1
   ```

2. **Implementación completa de instrucción READ:**
   ```go
   case "READ":
   	// ❌ ANTES: Código comentado
   	/*direccion, _ := strconv.Atoi(args[0])
   	tamanio, _ := strconv.Atoi(args[1])
   	dirFisica := traducirDireccion(pid, direccion)
   	datoLeido := h.readMemoria(pid, dirFisica, tamanio)*/
   	
   	// ✅ DESPUÉS: Implementación completa con módulo dedicado
   	if len(args) < 2 {
   		h.Log.Error("READ requiere al menos 2 argumentos")
   		return false, pc
   	}
   	direccion := args[0]
   	tamanio, err := strconv.Atoi(args[1])
   	dirFisica := direccion // TODO: implementar traducción
   	
   	datoLeido, err := h.Memoria.Read(pid, dirFisica, tamanio)
   	// Validación completa y manejo de errores
   	nuevoPC = pc + 1
   ```
   
---

### **2. NUEVO MÓDULO: `cpu/pkg/memoria/`**

#### **📁 Archivo:** `cpu/pkg/memoria/memoria.go` (CREADO)

**🔧 Estructura implementada:**

```go
// Estructura principal
type Memoria struct {
    IP     string
    Puerto int
    Log    *slog.Logger
}

// Estructuras de datos
type PeticionAcceso struct {
    PID       int    `json:"pid"`
    Direccion string `json:"direccion"`
    Datos     string `json:"datos,omitempty"`     // Solo para WRITE
    Tamanio   int    `json:"tamanio,omitempty"`   // Solo para READ
    Operacion string `json:"operacion"`           // "READ" o "WRITE"
}

type RespuestaAcceso struct {
    Datos   string `json:"datos,omitempty"`   // Solo para read
    Exito   bool   `json:"exito"`
    Mensaje string `json:"mensaje,omitempty"`
}

type PeticionInstruccion struct {
    PID int `json:"pid"`
    PC  int `json:"pc"`
}

type Instruccion struct {
    Instruccion string   `json:"instruccion"`
    Parametros  []string `json:"parametros"`
}
```

**🎯 Métodos implementados:**

1. **Constructor:**
   ```go
   func NewMemoria(ip string, puerto int, logger *slog.Logger) *Memoria
   ```

2. **Operaciones de memoria:**
   ```go
   func (m *Memoria) Write(pid int, direccion string, datos string) error
   func (m *Memoria) Read(pid int, direccion string, tamanio int) (string, error)
   func (m *Memoria) FetchInstruccion(pid int, pc int) (Instruccion, error)
   ```

---

### **3. Handler CPU Actualizado (`cpu/cmd/api/handler.go`)**

**🔧 Cambios realizados:**

```go
// ❌ ANTES
type Handler struct {
    Log     *slog.Logger
    Config  *Config
    Service *internal.Service
}

// ✅ DESPUÉS
type Handler struct {
    Log     *slog.Logger
    Config  *Config
    Service *internal.Service
    Memoria *memoria.Memoria  // ← Nuevo cliente de memoria
}
```

**Inicialización automática:**
```go
return &Handler{
    Config:  configStruct,
    Log:     logger,
    Service: internal.NewService(logger, configStruct.IpKernel, configStruct.PortKernel),
    Memoria: memoria.NewMemoria(configStruct.IpMemory, configStruct.PortMemory, logger),
}
```

---

### **4. Módulo Memoria - Endpoint CPU Mejorado**

#### **📁 Archivo:** `memoria/cmd/api/acceso.go`

**🔧 Cambios realizados:**

1. **Nuevas estructuras de datos:**
   ```go
   type PeticionAcceso struct {
   	PID       int    `json:"pid"`
   	Direccion string `json:"direccion"`
   	Datos     string `json:"datos,omitempty"`     // Solo para WRITE
   	Tamanio   int    `json:"tamanio,omitempty"`   // Solo para READ
   	Operacion string `json:"operacion"`           // "READ" o "WRITE"
   }
   
   type RespuestaAcceso struct {
   	Datos   string `json:"datos,omitempty"`   // Solo para read
   	Exito   bool   `json:"exito"`
   	Mensaje string `json:"mensaje,omitempty"`
   }
   ```

2. **Función RecibirPeticionAcceso completamente reescrita:**
   ```go
   switch peticion.Operacion {
   case "READ":
   	// Simulación de lectura con datos mockeados
   	datosMock := fmt.Sprintf("valor_en_%s_pid_%d", peticion.Direccion, peticion.PID)
   	// Respuesta estructurada JSON
   	
   case "WRITE":
   	// Simulación de escritura
   	// Logging detallado
   }
   ```

3. **Delay de memoria configurable:**
   ```go
   if h.Config.MemoryDelay > 0 {
   	time.Sleep(time.Duration(h.Config.MemoryDelay) * time.Millisecond)
   }
   ```

#### **📁 Archivo:** `memoria/memoria.go`

**🔧 Nuevo endpoint agregado:**
```go
mux.HandleFunc("POST /cpu/acceso", h.RecibirPeticionAcceso) // CPU --> Memoria (READ/WRITE)
```

---

### **5. Archivos de Prueba y Documentación**

#### **📁 Archivo:** `memoria/examples/proceso_test` (CREADO)

**🔧 Contenido de prueba:**
```
NOOP
WRITE 100 Hola_Mundo
READ 100 4
NOOP
WRITE 200 Test_Checkpoint2
READ 200 15
GOTO 8
NOOP
IO 5000
EXIT
```
