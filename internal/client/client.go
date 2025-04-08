package client

import (
	pb "Gault/gen/go/api/proto/v1"
	"fmt"

	"github.com/rivo/tview"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// maxSizeBytes максимальный размер файла
const maxSizeBytes = 1024 * 1024 * 1024 * 100 // 100GB

var (
	// pages страницы TUI
	pages *tview.Pages
	// autClient клиент авторизации
	autClient pb.AuthV1ServiceClient
	// dataClient клиент работы с данными
	dataClient pb.ContentManagerV1ServiceClient
)

// GrpcClient устанавливает gRPC-соединение
func GrpcClient(port int) (*grpc.ClientConn, error) {
	conn, err := grpc.NewClient(
		fmt.Sprintf(":%d", port),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithDefaultCallOptions(
			grpc.MaxCallRecvMsgSize(maxSizeBytes),
			grpc.MaxCallSendMsgSize(maxSizeBytes),
		),
	)
	if err != nil {
		return nil, err
	}
	autClient = pb.NewAuthV1ServiceClient(conn)
	dataClient = pb.NewContentManagerV1ServiceClient(conn)
	return conn, nil
}

// TUIClientWithApp запуск TUI
func TUIClientWithApp(app *tview.Application) error {
	pages = tview.NewPages()

	loginFlex := showLoginMenu(app)
	pages.AddPage("login", loginFlex, true, true)

	app.SetRoot(pages, true).SetFocus(loginFlex)
	return app.Run()
}
