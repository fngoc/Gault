package config

import (
	"Gault/pkg/logger"
	"fmt"

	"github.com/spf13/viper"
)

// Config структура файла конфигурации
type Config struct {
	Port           int            `mapstructure:"port" default:"8080"`
	DB             string         `mapstructure:"db" default:"host=localhost user=postgres password=postgres dbname=test_db sslmode=disable"`
	AllowEndpoints []EndpointRule `mapstructure:"allowEndpoints"`
}

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
		return Config{}, fmt.Errorf("failed to read config: %w", err)
	}

	if err := viper.Unmarshal(&conf); err != nil {
		return Config{}, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	logger.LogInfo("loaded config")
	return conf, nil
}
