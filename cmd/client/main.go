package main

import (
	"Gault/cmd/client/ui"
	"Gault/internal/config"
	wire "Gault/internal/injector"
	"Gault/pkg/logger"
	"fmt"
	"log"

	"google.golang.org/grpc"
)

var (
	Version   = "dev"
	BuildDate = "unknown"
)

func main() {
	err := wire.InitializeLogger()
	if err != nil {
		log.Fatalf("failed to init logger: %v", err)
	}

	logger.LogInfo("Starting client")
	logger.LogInfo(fmt.Sprintf("Version: %s, Build date: %s", Version, BuildDate))

	conf, err := config.ParseConfig("client_config")
	if err != nil {
		logger.LogFatal(err.Error())
	}

	var conn *grpc.ClientConn
	go func() {
		var err error
		conn, err = ui.GrpcClient(conf.Port)
		if err != nil {
			logger.LogFatal(err.Error())
		}
	}()

	if conn != nil {
		defer conn.Close()
	}

	if err := ui.TUIClient(); err != nil {
		logger.LogFatal(err.Error())
	}
}
