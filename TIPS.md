# 🎯 Guía de Pruebas y Conocimientos

Este archivo contiene instrucciones detalladas sobre los pasos a seguir para ejecutar la aplicación de manera correcta,
asegurando que todos los módulos funcionen correctamente y cumplan con los requerimientos establecidos.

## 🚀 Cómo Probar

### **1. Correr Todos los Módulos (En diferentes Tabs de la terminal)**

**Terminal 1 - Memoria:**

```bash
cd memoria && go run memoria.go
```

**Terminal 2 - Kernel:**
Su argumento es el directorio de almacenamiento de las intrucciones del proceso a ejecutar y
el tamaño de la memoria asignada al proceso.

```bash
cd kernel && go run kernel.go examples\proceso1 10
```

**Terminal 3 - CPU:**
Su argumento es el puerto en el que escuchará las peticiones de Memoria.

```bash
cd cpu && go run cpu.go 8004
```

**Terminal 4 - IO:**
Su argumento es el nombre del dispositivo IO que se utilizará (por ejemplo, `impresora`).

```bash
cd io && go run io.go impresora
```



### **2. Archivos de Prueba Disponibles**

- `memoria/examples/proceso1` - Archivo de prueba con el ejemplo del enunciado.
- `memoria/examples/proceso_test` - Archivo de prueba nuevo con foco en READ/WRITE

**Contenido del _proceso1_:**
```
NOOP
WRITE 0 EJEMPLO_DE_ENUNCIADO
READ 0 20
GOTO 0
IO 25000
INIT_PROC proceso1 256
DUMP_MEMORY
EXIT
```

**Contenido del _proceso\_test_:**
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

## 💡 Notas Importantes

1. **Traducción de Direcciones**: Por ahora se usa la dirección lógica directamente (sin MMU) para cumplir con el checkpoint básico.

2. **Orden de Ejecución**: Es importante iniciar los módulos en el orden especificado para evitar errores de conexión.

3**Logs de Debug**: Configurar `log_level: "DEBUG"` en los archivos de configuración para ver información detallada.

## 📚 Cómo actualizar las dependencias
Para actualizar las dependencias del proyecto, ejecuta el siguiente comando en la raíz del proyecto:

```bash
go work sync
```

