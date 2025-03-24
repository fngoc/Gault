package logger

import (
	"go.uber.org/zap"
)

// Log будет доступен всему коду как синглтон.
// По умолчанию установлен no-op-логер, который не выводит никаких сообщений.
var Log = zap.NewNop()

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
	Log = zl
	return nil
}
