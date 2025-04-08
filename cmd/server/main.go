package main

import (
	"Gault/internal/config"
	"Gault/internal/db"
	wire "Gault/internal/injector"
	"Gault/internal/server"
	"log"
)

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

// run запуск сервера
func run() error {
	err := wire.InitializeLogger()
	if err != nil {
		return err
	}

	conf, err := config.ParseConfig("server_config")
	if err != nil {
		return err
	}

	store, err := db.InitializePostgresDB(conf.DB)
	if err != nil {
		return err
	}

	if err = server.Run(conf.Port, conf.AllowEndpoints, store); err != nil {
		return err
	}
	return nil
}
