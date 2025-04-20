package main

import (
	"flag"
	"fmt"
	"log"

	"github.com/fngoc/gault/internal/client"
	"github.com/fngoc/gault/internal/config"
	wire "github.com/fngoc/gault/internal/injector"
	"github.com/fngoc/gault/pkg/logger"

	"github.com/rivo/tview"
	"google.golang.org/grpc"
)

var (
	Version   = "dev"
	BuildDate = "unknown"
)

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

// run запуск клиента
func run() error {
	versionFlag := flag.Bool("version", false, "Print version and exit")
	flag.Parse()

	if *versionFlag {
		fmt.Printf("Version: %s\nBuild date: %s\n", Version, BuildDate)
		return nil
	}

	err := wire.InitializeLogger()
	if err != nil {
		return err
	}

	logger.LogInfo("Starting client")
	logger.LogInfo(fmt.Sprintf("Version: %s, Build date: %s", Version, BuildDate))

	conf, err := config.ParseConfig("client_config")
	if err != nil {
		return err
	}

	var conn *grpc.ClientConn
	go func() {
		var err error
		conn, err = client.GrpcClient(conf.Port)
		if err != nil {
			logger.LogFatal(err.Error())
		}
	}()

	if conn != nil {
		defer conn.Close()
	}

	app := tview.NewApplication()
	if err = client.TUIClientWithApp(app, conf.Aes); err != nil {
		return err
	}
	return nil
}
