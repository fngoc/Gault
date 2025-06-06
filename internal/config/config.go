package config

import (
	"fmt"

	"github.com/fngoc/gault/pkg/logger"

	"github.com/spf13/viper"
)

// Config структура файла конфигурации
type Config struct {
	Port           int            `mapstructure:"port" default:"8080"`
	Aes            string         `mapstructure:"aes" default:"00000000000000000000000000000000"`
	DB             string         `mapstructure:"db" default:"host=localhost user=postgres password=postgres dbname=test_db sslmode=disable"`
	AllowEndpoints []EndpointRule `mapstructure:"allowEndpoints"`
}

// EndpointRule доступность ручек
type EndpointRule struct {
	Path    string `mapstructure:"path"`
	Allowed bool   `mapstructure:"allowed"`
}

// ParseConfig парсинг конфигурации
func ParseConfig(nameConfig string) (Config, error) {
	var conf Config
	viper.SetConfigName(nameConfig)
	viper.SetConfigType("yml")
	viper.AddConfigPath(".")

	if err := viper.ReadInConfig(); err != nil {
		logger.LogInfo("config not found, using defaults port [8080], DB config and allow Login/Registration endpoints")
		return Config{
			Port: 8080,
			Aes:  "00000000000000000000000000000000",
			DB:   "host=localhost user=postgres password=postgres dbname=test_db sslmode=disable",
			AllowEndpoints: []EndpointRule{
				{Path: "/api.proto.v1.AuthV1Service/Login", Allowed: true},
				{Path: "/api.proto.v1.AuthV1Service/Registration", Allowed: true},
			},
		}, nil
	}

	if err := viper.Unmarshal(&conf); err != nil {
		return Config{}, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	logger.LogInfo("loaded config")
	return conf, nil
}
