package server

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io"
	"net"
	"os"

	"github.com/fngoc/gault/internal/config"
	"github.com/fngoc/gault/internal/db"
	"github.com/fngoc/gault/pkg/logger"
	"github.com/fngoc/gault/pkg/utils"

	"google.golang.org/grpc/credentials"

	"github.com/google/uuid"

	"google.golang.org/grpc/metadata"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	pb "github.com/fngoc/gault/gen/go/api/proto/v1"
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
		return nil, status.Errorf(codes.PermissionDenied, "login failed, not valid credentials")
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

// SaveData метод сохранения данных через streaming, используя Large Objects в Postgres
func (g *GaultService) SaveData(stream pb.ContentManagerV1Service_SaveDataServer) error {
	ctx := stream.Context()

	logger.LogInfo("SaveData: starting transaction")
	tx, err := g.rep.BeginTx(ctx)
	if err != nil {
		return status.Errorf(codes.Internal, "begin tx failed: %v", err)
	}

	// Создаём пустой Large Object
	oid, err := g.rep.CreateEmptyLO(ctx, tx)
	if err != nil {
		return status.Errorf(codes.Internal, "CreateEmptyLO failed: %v", err)
	}
	logger.LogInfo(fmt.Sprintf("Created empty Large Object with OID: %d", oid))

	// Открываем Large Object для записи
	fd, err := g.rep.OpenLOForWriting(ctx, tx, oid)
	if err != nil {
		return status.Errorf(codes.Internal, "OpenLOForWriting failed: %v", err)
	}
	defer func() {
		logger.LogInfo(fmt.Sprintf("Closing Large Object FD: %d", fd))
		g.rep.CloseLO(ctx, tx, fd)
	}()

	var (
		userUID     string
		dataType    string
		dataName    string
		recordID    = uuid.New().String()
		recordReady bool
		chunkCount  uint64
		totalBytes  uint64
	)

	logger.LogInfo("Start receiving chunks from client")
	for {
		req, recvErr := stream.Recv()
		if recvErr == io.EOF {
			logger.LogInfo("Reached end of stream (EOF)")
			break
		}
		if recvErr != nil {
			return status.Errorf(codes.Internal, "receive chunk error: %v", recvErr)
		}

		// При первом чанке создаем запись в user_data
		if !recordReady {
			userUID = req.GetUserUid()
			dataType = req.GetType()
			dataName = req.GetName()

			logger.LogInfo(fmt.Sprintf("Creating user_data record: ID=%s, UserUID=%s, Type=%s, Name=%s, OID=%d",
				recordID, userUID, dataType, dataName, oid))

			if err := g.rep.InsertUserDataRecordTx(ctx, tx, recordID, userUID, dataType, dataName, oid); err != nil {
				return status.Errorf(codes.Internal, "insert user_data failed: %v", err)
			}
			recordReady = true
		}

		chunk := req.GetData()
		if len(chunk) > 0 {
			chunkCount++
			totalBytes += uint64(len(chunk))
			logger.LogInfo(fmt.Sprintf("Writing chunk #%d, size=%d bytes (total so far: %d bytes)",
				chunkCount, len(chunk), totalBytes))

			if err := g.rep.WriteLO(ctx, tx, fd, chunk); err != nil {
				return status.Errorf(codes.Internal, "failed writing chunk %d: %v", chunkCount, err)
			}
		}
	}

	if !recordReady {
		return status.Errorf(codes.InvalidArgument, "no data received")
	}

	logger.LogInfo(fmt.Sprintf("All chunks received. Total chunks: %d, total bytes: %d", chunkCount, totalBytes))

	if err = tx.Commit(); err != nil {
		return status.Errorf(codes.Internal, "commit failed: %v", err)
	}

	logger.LogInfo("Transaction committed successfully. Sending response to client.")
	return stream.SendAndClose(&pb.SaveDataResponse{})
}

// DeleteData метод удаления данных GaultService
func (g *GaultService) DeleteData(ctx context.Context, req *pb.DeleteDataRequest) (*pb.DeleteDataResponse, error) {
	if err := g.rep.DeleteData(ctx, req.GetId()); err != nil {
		return nil, err
	}
	return &pb.DeleteDataResponse{}, nil
}

// UpdateData метод обновления данных GaultService
func (g *GaultService) UpdateData(stream pb.ContentManagerV1Service_UpdateDataServer) error {
	ctx := stream.Context()

	logger.LogInfo("UpdateData: starting transaction")
	tx, err := g.rep.BeginTx(ctx)
	if err != nil {
		return status.Errorf(codes.Internal, "begin tx failed: %v", err)
	}

	firstReq, recvErr := stream.Recv()
	if recvErr == io.EOF {
		return status.Errorf(codes.InvalidArgument, "no data received")
	}
	if recvErr != nil {
		return status.Errorf(codes.Internal, "receive chunk error: %v", recvErr)
	}

	// Ищем OID в уже существующей записи
	oid, err := g.rep.GetOidByItemID(ctx, firstReq.GetDataUid())
	if err != nil {
		return status.Errorf(codes.Internal, "GetOidByItemID failed: %v", err)
	}
	logger.LogInfo(fmt.Sprintf("Updating existing OID = %d", oid))

	// Открываем LO на запись
	fd, err := g.rep.OpenLOForWriting(ctx, tx, oid)
	if err != nil {
		return status.Errorf(codes.Internal, "OpenLOForWriting failed: %v", err)
	}
	defer func() {
		logger.LogInfo(fmt.Sprintf("Closing Large Object FD: %d", fd))
		g.rep.CloseLO(ctx, tx, fd)
	}()

	// Обнуляем содержимое
	if errTrunc := g.rep.TruncateLO(ctx, tx, fd, 0); errTrunc != nil {
		return status.Errorf(codes.Internal, "truncate LO failed: %v", errTrunc)
	}
	logger.LogInfo("LO truncated to 0 bytes")

	// Записываем первый чанк (который уже прочитали)
	var chunkCount uint64
	var totalBytes uint64

	data := firstReq.GetData()
	if len(data) > 0 {
		chunkCount++
		totalBytes += uint64(len(data))
		logger.LogInfo(fmt.Sprintf("Writing first chunk #%d: size=%d bytes", chunkCount, len(data)))

		if err := g.rep.WriteLO(ctx, tx, fd, data); err != nil {
			return status.Errorf(codes.Internal, "failed writing chunk %d: %v", chunkCount, err)
		}
	}

	logger.LogInfo("Start receiving subsequent chunks for UpdateData")

	// Читаем остальные чанки в цикле
	for {
		req, recvErr := stream.Recv()
		if recvErr == io.EOF {
			logger.LogInfo("Reached end of stream for update")
			break
		}
		if recvErr != nil {
			return status.Errorf(codes.Internal, "receive chunk error: %v", recvErr)
		}

		chunk := req.GetData()
		if len(chunk) > 0 {
			chunkCount++
			totalBytes += uint64(len(chunk))
			logger.LogInfo(fmt.Sprintf("Writing chunk #%d, size=%d bytes (total so far: %d)",
				chunkCount, len(chunk), totalBytes))

			if err := g.rep.WriteLO(ctx, tx, fd, chunk); err != nil {
				return status.Errorf(codes.Internal, "failed writing chunk %d: %v", chunkCount, err)
			}
		}
	}

	if chunkCount == 0 {
		return status.Errorf(codes.InvalidArgument, "no data to update")
	}

	logger.LogInfo(fmt.Sprintf("All chunks received for update, total chunks=%d, total bytes=%d", chunkCount, totalBytes))
	if err := tx.Commit(); err != nil {
		return status.Errorf(codes.Internal, "commit failed: %v", err)
	}

	logger.LogInfo("UpdateData: transaction committed, sending response")
	return stream.SendAndClose(&pb.UpdateDataResponse{})
}

// gaultServer инстанс сервиса
var gaultServer *GaultService

// Run запуск сервиса
func Run(port int, unprotectedMethods []config.EndpointRule, store db.Repository) error {
	listen, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		return err
	}

	// Загрузка TLS-сертификата и ключа
	cert, err := tls.LoadX509KeyPair("certs/server.crt", "certs/server.key")
	if err != nil {
		return fmt.Errorf("failed to load server cert/key: %w", err)
	}

	// Подгружаем CA для валидации клиента
	certPool := x509.NewCertPool()
	ca, err := os.ReadFile("certs/ca.crt")
	if err != nil {
		return fmt.Errorf("failed to read CA cert: %w", err)
	}
	certPool.AppendCertsFromPEM(ca)

	// Настройки TLS
	creds := credentials.NewTLS(&tls.Config{
		Certificates: []tls.Certificate{cert},
		ClientAuth:   tls.NoClientCert,
		ClientCAs:    certPool,
	})

	// Параметры gRPC-сервера
	serverOptions := []grpc.ServerOption{
		grpc.Creds(creds),
		grpc.UnaryInterceptor(AuthInterceptor),
		grpc.MaxRecvMsgSize(1024 * 1024 * 1024 * 100),
		grpc.MaxSendMsgSize(1024 * 1024 * 1024 * 100),
	}

	s := grpc.NewServer(serverOptions...)
	gaultServer = &GaultService{rep: store}
	setAllowEndpoints(unprotectedMethods)

	pb.RegisterAuthV1ServiceServer(s, gaultServer)
	pb.RegisterContentManagerV1ServiceServer(s, gaultServer)

	logger.LogInfo("start gRPC server with TLS")
	if err = s.Serve(listen); err != nil {
		return err
	}
	return nil
}
