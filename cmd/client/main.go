package main

import (
	"Gault/cmd/client/ui"
	"Gault/internal/config"
	"Gault/pkg/logger"
	"fmt"

	"google.golang.org/grpc"
)

var (
	Version   = "dev"
	BuildDate = "unknown"
)

func main() {
	if err := logger.Initialize(); err != nil {
		panic(err)
	}

	logger.Log.Info("Starting client")
	logger.Log.Info(fmt.Sprintf("Version: %s, Build date: %s", Version, BuildDate))

	conf, err := config.ParseConfig("client_config")
	if err != nil {
		logger.Log.Fatal(err.Error())
	}

	var conn *grpc.ClientConn
	go func() {
		var err error
		conn, err = ui.GrpcClient(conf.Port)
		if err != nil {
			logger.Log.Fatal(err.Error())
		}
	}()

	if conn != nil {
		defer conn.Close()
	}

	if err := ui.TUIClient(); err != nil {
		logger.Log.Fatal(err.Error())
	}
}
