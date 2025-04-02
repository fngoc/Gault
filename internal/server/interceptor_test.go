package server

import (
	"Gault/internal/config"
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

func TestAuthInterceptor_UnprotectedMethod(t *testing.T) {
	ctx := context.Background()
	info := &grpc.UnaryServerInfo{FullMethod: "/api.proto.v1.AuthService/Login"}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return "success", nil
	}

	setAllowEndpoints([]config.EndpointRule{{Path: "/api.proto.v1.AuthService/Login", Allowed: true}})
	resp, err := AuthInterceptor(ctx, nil, info, handler)
	assert.NoError(t, err)
	assert.Equal(t, "success", resp)
}

func TestAuthInterceptor_NoMetadata(t *testing.T) {
	ctx := context.Background()
	info := &grpc.UnaryServerInfo{FullMethod: "/proto.gault.v1.ProtectedService/SomeMethod"}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return "success", nil
	}

	resp, err := AuthInterceptor(ctx, nil, info, handler)
	assert.Nil(t, resp)
	assert.Equal(t, codes.Unauthenticated, status.Code(err))
}

func TestAuthInterceptor_MissingToken(t *testing.T) {
	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs("useruid", "123"))
	info := &grpc.UnaryServerInfo{FullMethod: "/proto.gault.v1.ProtectedService/SomeMethod"}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return "success", nil
	}

	resp, err := AuthInterceptor(ctx, nil, info, handler)
	assert.Nil(t, resp)
	assert.Equal(t, codes.Unauthenticated, status.Code(err))
}

func TestAuthInterceptor_MissingUserUID(t *testing.T) {
	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs("authorization", "123"))
	info := &grpc.UnaryServerInfo{FullMethod: "/proto.gault.v1.ProtectedService/SomeMethod"}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return "success", nil
	}

	resp, err := AuthInterceptor(ctx, nil, info, handler)
	assert.Nil(t, resp)
	assert.Equal(t, codes.Unauthenticated, status.Code(err))
}
