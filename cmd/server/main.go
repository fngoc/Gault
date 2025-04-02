package main

import (
	"Gault/internal/config"
	"Gault/internal/db"
	"Gault/internal/server"
	"Gault/pkg/logger"
)

func main() {
	if err := logger.Initialize(); err != nil {
		panic(err)
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
