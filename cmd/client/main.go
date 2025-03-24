package main

import (
	"Gault/internal/client"
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
		conn, err = client.GrpcClient(conf.Port)
		if err != nil {
			logger.Log.Fatal(err.Error())
		}
	}()

	if conn != nil {
		defer conn.Close()
	}

	if err := client.TUIClient(); err != nil {
		logger.Log.Fatal(err.Error())
	}
}
