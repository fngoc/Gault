package logger

import (
	"go.uber.org/zap"
)

// Log будет доступен всему коду как синглтон
// По умолчанию установлен no-op-логер, который не выводит никаких сообщений
var log = zap.NewNop()

// loglevel уровень логирования по умолчанию
const loglevel = "INFO"

// NewLogger инициализирует синглтон логера с необходимым уровнем логирования
func NewLogger() error {
	lvl, err := zap.ParseAtomicLevel(loglevel)
	if err != nil {
		return err
	}
	cfg := zap.NewProductionConfig()
	cfg.Level = lvl
	zl, err := cfg.Build()
	if err != nil {
		return err
	}
	log = zl
	return nil
}

// LogInfo логирование с уровнем info
func LogInfo(msg string) {
	log.Info(msg)
}

// LogFatal логирование с завершением программы
func LogFatal(msg string) {
	log.Fatal(msg)
}
