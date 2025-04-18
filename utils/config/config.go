package config

import (
	"encoding/json"
	"log/slog"
	"os"
)

type Config struct {
	PortCpu          int    `json:"port_cpu"`
	IpCpu            string `json:"ip_cpu"`
	IpMemory         string `json:"ip_memory"`
	PortMemory       int    `json:"port_memory"`
	IpKernel         string `json:"ip_kernel"`
	PortKernel       int    `json:"port_kernel"`
	TlbEntries       int    `json:"tlb_entries"`
	TlbReplacement   string `json:"tlb_replacement"`
	CacheEntries     int    `json:"cache_entries"`
	CacheReplacement string `json:"cache_replacement"`
	CacheDelay       int    `json:"cache_delay"`
	LogLevel         string `json:"log_level"`
}

func IniciarConfiguracion(filePath string) *Config {
	var config *Config
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
