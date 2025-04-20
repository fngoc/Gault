//go:build wireinject
// +build wireinject

package injector

import (
	"Gault/pkg/logger"

	"github.com/google/wire"
)

func InitializeLogger() error {
	wire.Build(logger.NewLogger)
	return nil
}
