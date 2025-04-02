package logger

import (
	"go.uber.org/zap"
)

// Log будет доступен всему коду как синглтон.
// По умолчанию установлен no-op-логер, который не выводит никаких сообщений.
var log = zap.NewNop()

// loglevel уровень логирования по умолчанию
const loglevel = "INFO"

// Initialize инициализирует синглтон логера с необходимым уровнем логирования.
func Initialize() error {
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

func LogInfo(msg string) {
	log.Info(msg)
}

func LogWarn(msg string) {
	log.Warn(msg)
}

func LogFatal(msg string) {
	log.Fatal(msg)
}
