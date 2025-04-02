package main

import (
	"Gault/internal/config"
	"Gault/internal/db"
	wire "Gault/internal/injector"
	"Gault/internal/server"
	"Gault/pkg/logger"
	"log"
)

func main() {
	err := wire.InitializeLogger()
	if err != nil {
		log.Fatalf("failed to init logger: %v", err)
	}

	conf, err := config.ParseConfig("server_config")
	if err != nil {
		logger.LogFatal(err.Error())
	}

	store, err := db.InitializePostgresDB(conf.DB)
	if err != nil {
		logger.LogFatal(err.Error())
	}

	if err = server.Run(conf.Port, conf.AllowEndpoints, store); err != nil {
		logger.LogFatal(err.Error())
	}
}
