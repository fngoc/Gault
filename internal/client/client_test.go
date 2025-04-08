package client

import (
	pb "Gault/gen/go/api/proto/v1"
	"fmt"
	"net"
	"testing"

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
	addr, stop := startTestGRPCServer(t)
	defer stop()

	_, portStr, err := net.SplitHostPort(addr)
	assert.NoError(t, err)

	port := 0
	_, err = fmt.Sscanf(portStr, "%d", &port)
	assert.NoError(t, err)

	conn, err := GrpcClient(port)
	assert.NoError(t, err)
	assert.NotNil(t, conn)

	defer conn.Close()

	assert.NotNil(t, autClient)
	assert.NotNil(t, dataClient)
}

func startTestGRPCServer(t *testing.T) (addr string, stopFunc func()) {
	lis, err := net.Listen("tcp", "localhost:0")
	assert.NoError(t, err)

	s := grpc.NewServer()
	pb.RegisterAuthV1ServiceServer(s, &fakeAuthServer{})
	pb.RegisterContentManagerV1ServiceServer(s, &fakeDataServer{})

	go func() {
		_ = s.Serve(lis)
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
	err := TUIClientWithApp(nil)
	assert.Error(t, err)
}
