package client

import (
	pb "Gault/api/pb/api/proto"
	"fmt"

	"github.com/rivo/tview"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

var (
	pages *tview.Pages

	autClient  pb.AuthServiceClient
	dataClient pb.DataServiceClient
)

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

// TUIClient – основной вход в TUI. Создаёт приложение, pages и экран логина
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
