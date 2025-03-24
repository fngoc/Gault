package main

import (
	"Gault/internal/client"
	"Gault/internal/config"
	"Gault/pkg/logger"

	"google.golang.org/grpc"
)

func main() {
	if err := logger.Initialize(); err != nil {
		panic(err)
	}

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
