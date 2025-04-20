package client

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net"
	"os"
	"testing"

	pb "github.com/fngoc/gault/gen/go/api/proto/v1"

	"google.golang.org/grpc/credentials"

	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc"
)

type fakeAuthServer struct {
	pb.UnimplementedAuthV1ServiceServer
}

type fakeDataServer struct {
	pb.UnimplementedContentManagerV1ServiceServer
}

func TestGrpcClient_Success(t *testing.T) {
	addr, stop := startTestGRPCServerWithTLS(t)
	defer stop()

	_, portStr, err := net.SplitHostPort(addr)
	assert.NoError(t, err)

	port := 0
	_, err = fmt.Sscanf(portStr, "%d", &port)
	assert.NoError(t, err)

	_, err = GrpcClient(port)
	assert.Error(t, err)
}

func startTestGRPCServerWithTLS(t *testing.T) (addr string, stopFunc func()) {
	lis, err := net.Listen("tcp", "localhost:0")
	assert.NoError(t, err)

	// Загружаем TLS-сертификаты
	cert, err := tls.LoadX509KeyPair("../../certs/server.crt", "../../certs/server.key")
	assert.NoError(t, err)

	caCert, err := os.ReadFile("../../certs/ca.crt")
	assert.NoError(t, err)

	certPool := x509.NewCertPool()
	assert.True(t, certPool.AppendCertsFromPEM(caCert))

	creds := credentials.NewTLS(&tls.Config{
		Certificates: []tls.Certificate{cert},
		ClientAuth:   tls.NoClientCert,
		ClientCAs:    certPool,
	})

	s := grpc.NewServer(grpc.Creds(creds))
	pb.RegisterAuthV1ServiceServer(s, &fakeAuthServer{})
	pb.RegisterContentManagerV1ServiceServer(s, &fakeDataServer{})

	go func() {
		if err := s.Serve(lis); err != nil {
			t.Logf("server error: %v", err)
		}
	}()

	return lis.Addr().String(), s.Stop
}

func TestTUIClientWithApp(t *testing.T) {
	defer func() {
		err := recover()
		if err != nil {
			fmt.Printf("%+v\n", err)
		}
	}()
	err := TUIClientWithApp(nil, "RhBRyjuJvwmkvXFEohPIXGxKunGqohRM")
	assert.Error(t, err)
}
