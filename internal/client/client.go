package client

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"os"

	pb "github.com/fngoc/gault/gen/go/api/proto/v1"

	"google.golang.org/grpc/credentials"

	"github.com/rivo/tview"
	"google.golang.org/grpc"
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
	// Подгружаем CA
	certPool := x509.NewCertPool()
	ca, err := os.ReadFile("certs/ca.crt")
	if err != nil {
		return nil, fmt.Errorf("failed to read CA cert: %w", err)
	}
	certPool.AppendCertsFromPEM(ca)

	// TLS-конфигурация клиента
	creds := credentials.NewTLS(&tls.Config{
		RootCAs:            certPool,
		InsecureSkipVerify: false, // true — если самоподписанный, но лучше false
	})

	conn, err := grpc.NewClient(
		fmt.Sprintf(":%d", port),
		grpc.WithTransportCredentials(creds),
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
func TUIClientWithApp(app *tview.Application, aes string) error {
	pages = tview.NewPages()

	loginFlex := showLoginMenu(app, aes)
	pages.AddPage("login", loginFlex, true, true)

	app.SetRoot(pages, true).SetFocus(loginFlex)
	return app.Run()
}
