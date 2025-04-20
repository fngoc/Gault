//go:build wireinject
// +build wireinject

package injector

import (
	"github.com/fngoc/gault/pkg/logger"

	"github.com/google/wire"
)

func InitializeLogger() error {
	wire.Build(logger.NewLogger)
	return nil
}
