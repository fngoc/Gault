package main

import (
	"log"

	"github.com/fngoc/gault/internal/config"
	"github.com/fngoc/gault/internal/db"
	wire "github.com/fngoc/gault/internal/injector"
	"github.com/fngoc/gault/internal/server"
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
