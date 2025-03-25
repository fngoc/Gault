package main

import (
	pb "Gault/gen/go/api/proto/v1"
	"Gault/internal/config"
	"Gault/pkg/logger"
	"fmt"

	"github.com/rivo/tview"
	"google.golang.org/grpc/credentials/insecure"

	"google.golang.org/grpc"
)

var (
	Version   = "dev"
	BuildDate = "unknown"

	pages *tview.Pages

	autClient  pb.AuthServiceClient
	dataClient pb.DataServiceClient
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
		conn, err = GrpcClient(conf.Port)
		if err != nil {
			logger.Log.Fatal(err.Error())
		}
	}()

	if conn != nil {
		defer conn.Close()
	}

	if err := TUIClient(); err != nil {
		logger.Log.Fatal(err.Error())
	}
}

// GrpcClient устанавливает gRPC-соединение
func GrpcClient(port int) (*grpc.ClientConn, error) {
	conn, err := grpc.NewClient(
		fmt.Sprintf(":%d", port),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		return nil, err
	}
	autClient = pb.NewAuthServiceClient(conn)
	dataClient = pb.NewDataServiceClient(conn)
	return conn, nil
}

// TUIClient запуск TUI
func TUIClient() error {
	app := tview.NewApplication()
	pages = tview.NewPages()

	loginFlex := showLoginMenu(app)
	pages.AddPage("login", loginFlex, true, true)

	app.SetRoot(pages, true).SetFocus(loginFlex)

	if err := app.Run(); err != nil {
		return err
	}
	return nil
}
