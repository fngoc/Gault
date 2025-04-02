package server

import (
	"Gault/internal/config"
	"Gault/internal/db"
	"Gault/pkg/logger"
	"Gault/pkg/utils"
	"context"
	"fmt"
	"net"

	"google.golang.org/grpc/metadata"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	pb "Gault/gen/go/api/proto/v1"
)

// GaultService сервис взаимодействия с базой данных
type GaultService struct {
	pb.UnimplementedAuthV1ServiceServer
	pb.UnimplementedContentManagerV1ServiceServer
	rep db.Repository
}

// Login метод авторизации GaultService
func (g *GaultService) Login(ctx context.Context, req *pb.LoginRequest) (*pb.LoginResponse, error) {
	isCreate, err := g.rep.IsUserCreated(ctx, req.GetLogin())
	if err != nil {
		return nil, err
	}
	if !isCreate {
		return nil, status.Errorf(codes.PermissionDenied, "login failed")
	}

	userUID, token, err := g.rep.UpdateSessionUser(ctx, req.GetLogin(), req.GetPassword())
	if err != nil {
		return nil, err
	}

	return &pb.LoginResponse{Token: token, UserUid: userUID}, nil
}

// Registration метод регистрации GaultService
func (g *GaultService) Registration(ctx context.Context, req *pb.RegistrationRequest) (*pb.RegistrationResponse, error) {
	hash, err := utils.HashPassword(req.GetPassword())
	if err != nil {
		return nil, err
	}

	userUID, token, err := g.rep.CreateUser(ctx, req.GetLogin(), hash)
	if err != nil {
		return nil, err
	}

	return &pb.RegistrationResponse{Token: token, UserUid: userUID}, nil
}

// GetUserDataList метод получения листа информации данных GaultService
func (g *GaultService) GetUserDataList(ctx context.Context, req *pb.GetUserDataListRequest) (*pb.GetUserDataListResponse, error) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "metadata is not provided")
	}

	authUserUID, userUIDExists := md["useruid"]
	if !userUIDExists || len(authUserUID) == 0 {
		return nil, status.Error(codes.Unauthenticated, "useruid is not provided")
	}
	userUID := authUserUID[0]

	list, err := g.rep.GetDataNameList(ctx, userUID)
	if err != nil {
		return nil, err
	}
	return list, err
}

// GetData метод получения данных GaultService
func (g *GaultService) GetData(ctx context.Context, req *pb.GetDataRequest) (*pb.GetDataResponse, error) {
	data, err := g.rep.GetData(ctx, req.GetId())
	if err != nil {
		return nil, err
	}
	return data, nil
}

// SaveData метод сохранения данных GaultService
func (g *GaultService) SaveData(ctx context.Context, req *pb.SaveDataRequest) (*pb.SaveDataResponse, error) {
	if err := g.rep.SaveData(ctx, req.GetUserUid(), req.GetType(), req.GetName(), req.GetData()); err != nil {
		return nil, err
	}
	return &pb.SaveDataResponse{}, nil
}

// DeleteData метод удаления данных GaultService
func (g *GaultService) DeleteData(ctx context.Context, req *pb.DeleteDataRequest) (*pb.DeleteDataResponse, error) {
	if err := g.rep.DeleteData(ctx, req.GetId()); err != nil {
		return nil, err
	}
	return &pb.DeleteDataResponse{}, nil
}

// UpdateData метод обновления данных GaultService
func (g *GaultService) UpdateData(ctx context.Context, req *pb.UpdateDataRequest) (*pb.UpdateDataResponse, error) {
	if err := g.rep.UpdateData(ctx, req.GetId(), req.GetData()); err != nil {
		return nil, err
	}
	return &pb.UpdateDataResponse{}, nil
}

// gaultServer инстанс сервиса
var gaultServer *GaultService

// Run запуск сервиса
func Run(port int, unprotectedMethods []config.EndpointRule, store db.Repository) error {
	listen, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		return err
	}

	gaultServer = &GaultService{rep: store}

	setAllowEndpoints(unprotectedMethods)

	// Увеличенные лимиты для входящих/исходящих сообщений (100 MB)
	serverOptions := []grpc.ServerOption{
		grpc.UnaryInterceptor(AuthInterceptor),
		grpc.MaxRecvMsgSize(1024 * 1024 * 100),
		grpc.MaxSendMsgSize(1024 * 1024 * 100),
	}

	s := grpc.NewServer(serverOptions...)
	pb.RegisterAuthV1ServiceServer(s, gaultServer)
	pb.RegisterContentManagerV1ServiceServer(s, gaultServer)

	logger.LogInfo("start gRPC server")
	if err = s.Serve(listen); err != nil {
		return err
	}
	return nil
}
