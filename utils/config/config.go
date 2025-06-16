package config

import (
	"encoding/json"
	"log/slog"
	"os"
)

func IniciarConfiguracion(filePath string, config interface{}) interface{} {
	configFile, err := os.Open(filePath)
	if err != nil {
		slog.Error("Error al abrir el archivo de configuración",
			slog.Attr{Key: "filePath", Value: slog.StringValue(filePath)},
			slog.Attr{Key: "error", Value: slog.StringValue(err.Error())},
		)
		panic(err)
	}
	defer func() {
		_ = configFile.Close()
	}()

	jsonParser := json.NewDecoder(configFile)
	err = jsonParser.Decode(&config)
	if err != nil {
		slog.Error("Error al decodificar el archivo de configuración",
			slog.Attr{Key: "filePath", Value: slog.StringValue(filePath)},
			slog.Attr{Key: "error", Value: slog.StringValue(err.Error())},
		)
		panic(err)
	}

	return config
}
