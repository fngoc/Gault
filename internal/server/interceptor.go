package server

import (
	"Gault/pkg/logger"
	"context"
	"fmt"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

// unprotectedMethods список методов, которые НЕ требуют аутентификации
var unprotectedMethods = map[string]bool{
	"/proto.gault.v1.AuthService/Login":        true,
	"/proto.gault.v1.AuthService/Registration": true,
}

// AuthInterceptor проверяет токен сессии в каждом запросе
func AuthInterceptor(
	ctx context.Context,
	req interface{},
	info *grpc.UnaryServerInfo,
	handler grpc.UnaryHandler,
) (interface{}, error) {
	if unprotectedMethods[info.FullMethod] {
		return handler(ctx, req)
	}

	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "metadata is not provided")
	}

	authHeader, authExists := md["authorization"]
	if !authExists || len(authHeader) == 0 {
		return nil, status.Error(codes.Unauthenticated, "token is not provided")
	}
	authUserUID, userUIDExists := md["useruid"]
	if !userUIDExists || len(authUserUID) == 0 {
		return nil, status.Error(codes.Unauthenticated, "useruid is not provided")
	}
	token := authHeader[0]
	userUID := authUserUID[0]

	if !gaultServer.rep.CheckSessionUser(ctx, userUID, token) {
		return nil, status.Error(codes.Unauthenticated, "user is not authorized")
	}
	logger.Log.Info(fmt.Sprintf("%s %s", info.FullMethod, token))

	return handler(ctx, req)
}
