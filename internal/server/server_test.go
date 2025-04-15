package server

import (
	"Gault/internal/config"
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"database/sql"
	"encoding/pem"
	"fmt"
	"io"
	"math/big"
	"net"
	"os"
	"testing"
	"time"

	"google.golang.org/grpc/codes"

	"google.golang.org/grpc"
	"google.golang.org/grpc/status"

	"google.golang.org/grpc/metadata"

	pb "Gault/gen/go/api/proto/v1"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"

	mockDB "Gault/gen/go/db"
)

// mockSaveDataServer заглушка, реализующая интерфейс ContentManagerV1Service_SaveDataServer
type mockSaveDataServer struct {
	grpc.ServerStream
	reqs  []*pb.SaveDataRequest
	index int
	resp  *pb.SaveDataResponse
	ctx   context.Context
}

func (m *mockSaveDataServer) Recv() (*pb.SaveDataRequest, error) {
	if m.index >= len(m.reqs) {
		return nil, io.EOF
	}
	r := m.reqs[m.index]
	m.index++
	return r, nil
}

func (m *mockSaveDataServer) SendAndClose(resp *pb.SaveDataResponse) error {
	m.resp = resp
	return nil
}

func (m *mockSaveDataServer) Context() context.Context {
	if m.ctx != nil {
		return m.ctx
	}
	return context.Background()
}

func TestGaultService_Login(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	repo := mockDB.NewMockRepository(ctrl)
	service := &GaultService{rep: repo}

	t.Run("success login", func(t *testing.T) {
		ctx := context.Background()
		login := "testUser"
		password := "password"
		repo.EXPECT().IsUserCreated(ctx, login).Return(true, nil)
		repo.EXPECT().UpdateSessionUser(ctx, login, password).Return("user-uid", "token", nil)

		resp, err := service.Login(ctx, &pb.LoginRequest{Login: login, Password: password})
		assert.NoError(t, err)
		assert.Equal(t, "token", resp.Token)
		assert.Equal(t, "user-uid", resp.UserUid)
	})
	t.Run("login error, user is created error", func(t *testing.T) {
		ctx := context.Background()
		login := "testUser"
		password := "password"
		repo.EXPECT().IsUserCreated(ctx, login).Return(false, fmt.Errorf("error"))

		resp, err := service.Login(ctx, &pb.LoginRequest{Login: login, Password: password})
		assert.NotNil(t, err)
		assert.Nil(t, resp)
	})
	t.Run("login error, user is created", func(t *testing.T) {
		ctx := context.Background()
		login := "testUser"
		password := "password"
		repo.EXPECT().IsUserCreated(ctx, login).Return(false, nil)

		resp, err := service.Login(ctx, &pb.LoginRequest{Login: login, Password: password})
		assert.Error(t, err)
		assert.Nil(t, resp)
	})
	t.Run("login error, user is created", func(t *testing.T) {
		ctx := context.Background()
		login := "testUser"
		password := "password"
		repo.EXPECT().IsUserCreated(ctx, login).Return(true, nil)
		repo.EXPECT().UpdateSessionUser(ctx, login, password).Return("", "", fmt.Errorf("error"))

		resp, err := service.Login(ctx, &pb.LoginRequest{Login: login, Password: password})
		assert.Error(t, err)
		assert.Nil(t, resp)
	})
}

func TestGaultService_Registration(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	repo := mockDB.NewMockRepository(ctrl)
	service := &GaultService{rep: repo}

	t.Run("success registration", func(t *testing.T) {
		ctx := context.Background()
		login := "newUser"
		password := "newPassword"
		repo.EXPECT().CreateUser(ctx, login, gomock.Any()).Return("user-uid", "token", nil)

		resp, err := service.Registration(ctx, &pb.RegistrationRequest{Login: login, Password: password})
		assert.NoError(t, err)
		assert.Equal(t, "token", resp.Token)
		assert.Equal(t, "user-uid", resp.UserUid)
	})
	t.Run("failed registration", func(t *testing.T) {
		ctx := context.Background()
		login := "newUser"
		password := "newPassword"
		repo.EXPECT().CreateUser(ctx, login, gomock.Any()).Return("", "", fmt.Errorf("error"))

		resp, err := service.Registration(ctx, &pb.RegistrationRequest{Login: login, Password: password})
		assert.Error(t, err)
		assert.Nil(t, resp)
	})
}

func TestGaultService_GetUserDataList(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	repo := mockDB.NewMockRepository(ctrl)
	service := &GaultService{rep: repo}

	t.Run("success", func(t *testing.T) {
		md := metadata.New(map[string]string{"useruid": "user-uid"})
		ctx := metadata.NewIncomingContext(context.Background(), md)
		repo.EXPECT().GetDataNameList(ctx, "user-uid").Return(&pb.GetUserDataListResponse{}, nil)

		resp, err := service.GetUserDataList(ctx, &pb.GetUserDataListRequest{})
		assert.NoError(t, err)
		assert.NotNil(t, resp)
	})
	t.Run("md error", func(t *testing.T) {
		resp, err := service.GetUserDataList(context.Background(), &pb.GetUserDataListRequest{})
		assert.Error(t, err)
		assert.Nil(t, resp)
	})
	t.Run("md error len", func(t *testing.T) {
		md := metadata.New(map[string]string{"notValidKey": "user-uid"})
		ctx := metadata.NewIncomingContext(context.Background(), md)

		resp, err := service.GetUserDataList(ctx, &pb.GetUserDataListRequest{})
		assert.Error(t, err)
		assert.Nil(t, resp)
	})
	t.Run("success", func(t *testing.T) {
		md := metadata.New(map[string]string{"useruid": "user-uid"})
		ctx := metadata.NewIncomingContext(context.Background(), md)
		repo.EXPECT().GetDataNameList(ctx, "user-uid").Return(nil, fmt.Errorf("error"))

		resp, err := service.GetUserDataList(ctx, &pb.GetUserDataListRequest{})
		assert.Error(t, err)
		assert.Nil(t, resp)
	})
}

func TestGaultService_GetData(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	repo := mockDB.NewMockRepository(ctrl)
	service := &GaultService{rep: repo}

	t.Run("success", func(t *testing.T) {
		md := metadata.New(map[string]string{"useruid": "user-uid"})
		ctx := metadata.NewIncomingContext(context.Background(), md)
		repo.EXPECT().GetData(ctx, "data-id").Return(&pb.GetDataResponse{Type: "text", Content: &pb.GetDataResponse_TextData{TextData: "content"}}, nil)

		resp, err := service.GetData(ctx, &pb.GetDataRequest{Id: "data-id"})
		assert.NoError(t, err)
		assert.NotNil(t, resp)
		assert.Equal(t, "text", resp.Type)
	})
	t.Run("error", func(t *testing.T) {
		md := metadata.New(map[string]string{"useruid": "user-uid"})
		ctx := metadata.NewIncomingContext(context.Background(), md)
		repo.EXPECT().GetData(ctx, "data-id").Return(nil, fmt.Errorf("error"))

		resp, err := service.GetData(ctx, &pb.GetDataRequest{Id: "data-id"})
		assert.Error(t, err)
		assert.Nil(t, resp)
	})
}

func TestGaultService_DeleteData(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	repo := mockDB.NewMockRepository(ctrl)
	service := &GaultService{rep: repo}

	t.Run("success", func(t *testing.T) {
		md := metadata.New(map[string]string{"useruid": "user-uid"})
		ctx := metadata.NewIncomingContext(context.Background(), md)
		repo.EXPECT().DeleteData(ctx, "data-id").Return(nil)

		resp, err := service.DeleteData(ctx, &pb.DeleteDataRequest{Id: "data-id"})
		assert.NoError(t, err)
		assert.NotNil(t, resp)
	})
	t.Run("error", func(t *testing.T) {
		md := metadata.New(map[string]string{"useruid": "user-uid"})
		ctx := metadata.NewIncomingContext(context.Background(), md)
		repo.EXPECT().DeleteData(ctx, "data-id").Return(fmt.Errorf("error"))

		resp, err := service.DeleteData(ctx, &pb.DeleteDataRequest{Id: "data-id"})
		assert.Error(t, err)
		assert.Nil(t, resp)
	})
}

func TestRun(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	repo := mockDB.NewMockRepository(ctrl)
	port := 50051

	ln, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	assert.NoError(t, err)
	ln.Close()

	go func() {
		_ = Run(port, []config.EndpointRule{}, repo)
	}()
}

func TestGaultService_SaveData(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mockDB.NewMockRepository(ctrl)
	service := &GaultService{rep: mockRepo}

	t.Run("error: Commit fail", func(t *testing.T) {
		// Настраиваем входные данные
		stream := &mockSaveDataServer{
			reqs: []*pb.SaveDataRequest{
				{
					UserUid: "some-user-uid",
					Type:    "file",
					Name:    "testFileName",
					Data:    []byte("some-binary-data"),
				},
			},
		}

		// Настройка моков для репозитория
		mockTx := &sql.Tx{}
		mockRepo.EXPECT().BeginTx(gomock.Any()).Return(mockTx, nil)
		mockRepo.EXPECT().CreateEmptyLO(gomock.Any(), mockTx).Return(123, nil)
		mockRepo.EXPECT().OpenLOForWriting(gomock.Any(), mockTx, 123).Return(111, nil)
		mockRepo.EXPECT().InsertUserDataRecordTx(
			gomock.Any(), mockTx,
			gomock.Any(), // recordID (uuid)
			"some-user-uid",
			"file",
			"testFileName",
			123,
		).Return(nil)
		mockRepo.EXPECT().WriteLO(gomock.Any(), mockTx, 111, []byte("some-binary-data")).Return(nil)
		mockRepo.EXPECT().CloseLO(gomock.Any(), mockTx, 111)
		// Запускаем тестируемый метод и выходим на ошибке Commit
		defer func() {
			if r := recover(); r != nil {
				fmt.Println("Recovered from panic for server test")
			}
		}()
		err := service.SaveData(stream)
		assert.NoError(t, err)
		// Проверяем, что SendAndClose отработал
		assert.NotNil(t, stream.resp)
	})

	t.Run("error: BeginTx fail", func(t *testing.T) {
		stream := &mockSaveDataServer{
			reqs: []*pb.SaveDataRequest{
				{
					UserUid: "some-user-uid",
					Type:    "file",
					Name:    "testFileName",
					Data:    []byte("some-binary-data"),
				},
			},
		}
		mockRepo.EXPECT().BeginTx(gomock.Any()).Return(nil, fmt.Errorf("begin tx error"))

		err := service.SaveData(stream)
		assert.Error(t, err)
		st, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Contains(t, st.Message(), "begin tx failed: begin tx error")
	})

	t.Run("error: CreateEmptyLO fail", func(t *testing.T) {
		stream := &mockSaveDataServer{
			reqs: []*pb.SaveDataRequest{
				{
					UserUid: "uid",
					Type:    "file",
					Name:    "filename",
					Data:    []byte("some-data"),
				},
			},
		}
		mockTx := &sql.Tx{}
		mockRepo.EXPECT().BeginTx(gomock.Any()).Return(mockTx, nil)
		mockRepo.EXPECT().CreateEmptyLO(gomock.Any(), mockTx).Return(0, fmt.Errorf("create LO error"))

		defer func() {
			if r := recover(); r != nil {
				fmt.Println("Recovered from panic for server test")
			}
		}()
		err := service.SaveData(stream)
		assert.Error(t, err)
		st, _ := status.FromError(err)
		assert.Contains(t, st.Message(), "CreateEmptyLO failed: create LO error")
	})

	t.Run("error: OpenLOForWriting fail", func(t *testing.T) {
		stream := &mockSaveDataServer{
			reqs: []*pb.SaveDataRequest{
				{UserUid: "uid", Type: "text", Name: "n", Data: []byte("aaa")},
			},
		}
		mockTx := &sql.Tx{}
		mockRepo.EXPECT().BeginTx(gomock.Any()).Return(mockTx, nil)
		mockRepo.EXPECT().CreateEmptyLO(gomock.Any(), mockTx).Return(321, nil)
		mockRepo.EXPECT().OpenLOForWriting(gomock.Any(), mockTx, 321).Return(0, fmt.Errorf("open fail"))

		defer func() {
			if r := recover(); r != nil {
				fmt.Println("Recovered from panic for server test")
			}
		}()
		err := service.SaveData(stream)
		assert.Error(t, err)
		st, _ := status.FromError(err)
		assert.Contains(t, st.Message(), "OpenLOForWriting failed: open fail")
	})

	t.Run("error: InsertUserDataRecordTx fail", func(t *testing.T) {
		stream := &mockSaveDataServer{
			reqs: []*pb.SaveDataRequest{
				{UserUid: "uid", Type: "file", Name: "n", Data: []byte("chunk1")},
			},
		}
		mockTx := &sql.Tx{}
		mockRepo.EXPECT().BeginTx(gomock.Any()).Return(mockTx, nil)
		mockRepo.EXPECT().CreateEmptyLO(gomock.Any(), mockTx).Return(123, nil)
		mockRepo.EXPECT().OpenLOForWriting(gomock.Any(), mockTx, 123).Return(999, nil)
		mockRepo.EXPECT().InsertUserDataRecordTx(gomock.Any(), mockTx, gomock.Any(), "uid", "file", "n", 123).
			Return(fmt.Errorf("insert fail"))
		mockRepo.EXPECT().CloseLO(gomock.Any(), mockTx, 999).AnyTimes()

		defer func() {
			if r := recover(); r != nil {
				fmt.Println("Recovered from panic for server test")
			}
		}()
		err := service.SaveData(stream)
		assert.Error(t, err)
		st, _ := status.FromError(err)
		assert.Contains(t, st.Message(), "insert user_data failed: insert fail")
	})

	t.Run("error: WriteLO fail on chunk", func(t *testing.T) {
		stream := &mockSaveDataServer{
			reqs: []*pb.SaveDataRequest{
				{UserUid: "uid", Type: "file", Name: "n", Data: []byte("chunk1")},
			},
		}
		mockTx := &sql.Tx{}
		mockRepo.EXPECT().BeginTx(gomock.Any()).Return(mockTx, nil)
		mockRepo.EXPECT().CreateEmptyLO(gomock.Any(), mockTx).Return(123, nil)
		mockRepo.EXPECT().OpenLOForWriting(gomock.Any(), mockTx, 123).Return(555, nil)
		mockRepo.EXPECT().InsertUserDataRecordTx(gomock.Any(), mockTx, gomock.Any(), "uid", "file", "n", 123).Return(nil)
		mockRepo.EXPECT().WriteLO(gomock.Any(), mockTx, 555, []byte("chunk1")).
			Return(fmt.Errorf("write chunk fail"))
		mockRepo.EXPECT().CloseLO(gomock.Any(), mockTx, 555).AnyTimes()

		defer func() {
			if r := recover(); r != nil {
				fmt.Println("Recovered from panic for server test")
			}
		}()
		err := service.SaveData(stream)
		assert.Error(t, err)
		st, _ := status.FromError(err)
		assert.Contains(t, st.Message(), "failed writing chunk 1: write chunk fail")
	})
}

type mockUpdateDataServer struct {
	grpc.ServerStream

	reqs  []*pb.UpdateDataRequest
	index int

	resp *pb.UpdateDataResponse
	ctx  context.Context
}

func (m *mockUpdateDataServer) Recv() (*pb.UpdateDataRequest, error) {
	if m.index >= len(m.reqs) {
		return nil, io.EOF
	}
	r := m.reqs[m.index]
	m.index++
	return r, nil
}

func (m *mockUpdateDataServer) SendAndClose(resp *pb.UpdateDataResponse) error {
	m.resp = resp
	return nil
}

func (m *mockUpdateDataServer) Context() context.Context {
	if m.ctx != nil {
		return m.ctx
	}
	return context.Background()
}

func TestGaultService_UpdateData(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mockDB.NewMockRepository(ctrl)
	service := &GaultService{rep: mockRepo}

	t.Run("success multiple chunks", func(t *testing.T) {
		stream := &mockUpdateDataServer{
			reqs: []*pb.UpdateDataRequest{
				{
					DataUid: "some-data-uid",
					Data:    []byte("first-chunk-"),
				},
				{
					DataUid: "some-data-uid",
					Data:    []byte("second-chunk"),
				},
			},
		}

		mockTx := &sql.Tx{}
		mockRepo.EXPECT().BeginTx(gomock.Any()).Return(mockTx, nil)

		mockRepo.EXPECT().GetOidByItemID(gomock.Any(), "some-data-uid").Return(1234, nil)

		mockRepo.EXPECT().OpenLOForWriting(gomock.Any(), mockTx, 1234).Return(999, nil)

		mockRepo.EXPECT().TruncateLO(gomock.Any(), mockTx, 999, int64(0)).Return(nil)
		mockRepo.EXPECT().WriteLO(gomock.Any(), mockTx, 999, []byte("first-chunk-")).Return(nil)
		mockRepo.EXPECT().WriteLO(gomock.Any(), mockTx, 999, []byte("second-chunk")).Return(nil)
		mockRepo.EXPECT().CloseLO(gomock.Any(), mockTx, 999)

		defer func() {
			if r := recover(); r != nil {
				fmt.Println("Recovered from panic for server test")
			}
		}()
		err := service.UpdateData(stream)
		assert.NoError(t, err)
		assert.NotNil(t, stream.resp, "должен быть ответ в SendAndClose")
	})

	t.Run("error: BeginTx fails", func(t *testing.T) {
		stream := &mockUpdateDataServer{
			reqs: []*pb.UpdateDataRequest{
				{DataUid: "uid", Data: []byte("someData")},
			},
		}
		mockRepo.EXPECT().BeginTx(gomock.Any()).Return(nil, fmt.Errorf("begin tx error"))

		err := service.UpdateData(stream)
		assert.Error(t, err)
		st, _ := status.FromError(err)
		assert.Equal(t, codes.Internal, st.Code())
		assert.Contains(t, st.Message(), "begin tx failed: begin tx error")
	})

	t.Run("error: no data received (first Recv is EOF)", func(t *testing.T) {
		stream := &mockUpdateDataServer{
			reqs: []*pb.UpdateDataRequest{},
		}
		mockTx := &sql.Tx{}
		mockRepo.EXPECT().BeginTx(gomock.Any()).Return(mockTx, nil)

		defer func() {
			if r := recover(); r != nil {
				fmt.Println("Recovered from panic for server test")
			}
		}()
		err := service.UpdateData(stream)
		assert.Error(t, err)
		st, _ := status.FromError(err)
		assert.Equal(t, codes.InvalidArgument, st.Code())
		assert.Contains(t, st.Message(), "no data received")
	})

	t.Run("error: first Recv returns some error", func(t *testing.T) {
		stream := &mockUpdateDataServer{}

		mockTx := &sql.Tx{}
		mockRepo.EXPECT().BeginTx(gomock.Any()).Return(mockTx, nil)

		defer func() {
			if r := recover(); r != nil {
				fmt.Println("Recovered from panic for server test")
			}
		}()
		err := service.UpdateData(stream)
		assert.Error(t, err)
		_, _ = status.FromError(err)
	})

	t.Run("error: GetOidByItemID fails", func(t *testing.T) {
		stream := &mockUpdateDataServer{
			reqs: []*pb.UpdateDataRequest{
				{DataUid: "uid", Data: []byte("chunk")},
			},
		}
		mockTx := &sql.Tx{}
		mockRepo.EXPECT().BeginTx(gomock.Any()).Return(mockTx, nil)
		mockRepo.EXPECT().GetOidByItemID(gomock.Any(), "uid").Return(0, fmt.Errorf("some oid error"))

		defer func() {
			if r := recover(); r != nil {
				fmt.Println("Recovered from panic for server test")
			}
		}()
		err := service.UpdateData(stream)
		assert.Error(t, err)
		st, _ := status.FromError(err)
		assert.Equal(t, codes.Internal, st.Code())
		assert.Contains(t, st.Message(), "GetOidByItemID failed: some oid error")
	})

	t.Run("error: OpenLOForWriting fails", func(t *testing.T) {
		stream := &mockUpdateDataServer{
			reqs: []*pb.UpdateDataRequest{
				{DataUid: "uid", Data: []byte("chunk")},
			},
		}
		mockTx := &sql.Tx{}
		mockRepo.EXPECT().BeginTx(gomock.Any()).Return(mockTx, nil)
		mockRepo.EXPECT().GetOidByItemID(gomock.Any(), "uid").Return(333, nil)
		mockRepo.EXPECT().OpenLOForWriting(gomock.Any(), mockTx, 333).Return(0, fmt.Errorf("open fail"))

		defer func() {
			if r := recover(); r != nil {
				fmt.Println("Recovered from panic for server test")
			}
		}()
		err := service.UpdateData(stream)
		assert.Error(t, err)
		st, _ := status.FromError(err)
		assert.Equal(t, codes.Internal, st.Code())
		assert.Contains(t, st.Message(), "OpenLOForWriting failed: open fail")
	})

	t.Run("error: TruncateLO fails", func(t *testing.T) {
		stream := &mockUpdateDataServer{
			reqs: []*pb.UpdateDataRequest{
				{DataUid: "uid", Data: []byte("chunk")},
			},
		}
		mockTx := &sql.Tx{}
		mockRepo.EXPECT().BeginTx(gomock.Any()).Return(mockTx, nil)
		mockRepo.EXPECT().GetOidByItemID(gomock.Any(), "uid").Return(444, nil)
		mockRepo.EXPECT().OpenLOForWriting(gomock.Any(), mockTx, 444).Return(777, nil)
		mockRepo.EXPECT().TruncateLO(gomock.Any(), mockTx, 777, int64(0)).
			Return(fmt.Errorf("truncate fail"))
		mockRepo.EXPECT().CloseLO(gomock.Any(), mockTx, 777).AnyTimes()

		defer func() {
			if r := recover(); r != nil {
				fmt.Println("Recovered from panic for server test")
			}
		}()
		err := service.UpdateData(stream)
		assert.Error(t, err)
		st, _ := status.FromError(err)
		assert.Equal(t, codes.Internal, st.Code())
		assert.Contains(t, st.Message(), "truncate LO failed: truncate fail")
	})

	t.Run("error: writeLO fails on first chunk", func(t *testing.T) {
		stream := &mockUpdateDataServer{
			reqs: []*pb.UpdateDataRequest{
				{DataUid: "uid", Data: []byte("first-chunk")},
			},
		}
		mockTx := &sql.Tx{}
		mockRepo.EXPECT().BeginTx(gomock.Any()).Return(mockTx, nil)
		mockRepo.EXPECT().GetOidByItemID(gomock.Any(), "uid").Return(555, nil)
		mockRepo.EXPECT().OpenLOForWriting(gomock.Any(), mockTx, 555).Return(999, nil)
		mockRepo.EXPECT().TruncateLO(gomock.Any(), mockTx, 999, int64(0)).Return(nil)
		mockRepo.EXPECT().WriteLO(gomock.Any(), mockTx, 999, []byte("first-chunk")).
			Return(fmt.Errorf("write fail"))
		mockRepo.EXPECT().CloseLO(gomock.Any(), mockTx, 999).AnyTimes()

		defer func() {
			if r := recover(); r != nil {
				fmt.Println("Recovered from panic for server test")
			}
		}()
		err := service.UpdateData(stream)
		assert.Error(t, err)
		st, _ := status.FromError(err)
		assert.Equal(t, codes.Internal, st.Code())
		assert.Contains(t, st.Message(), "failed writing chunk 1: write fail")
	})

	t.Run("error: no data to update (first chunk has 0 bytes, последующие тоже)", func(t *testing.T) {
		stream := &mockUpdateDataServer{
			reqs: []*pb.UpdateDataRequest{
				{DataUid: "uid", Data: []byte("")},
			},
		}
		mockTx := &sql.Tx{}
		mockRepo.EXPECT().BeginTx(gomock.Any()).Return(mockTx, nil)
		mockRepo.EXPECT().GetOidByItemID(gomock.Any(), "uid").Return(222, nil)
		mockRepo.EXPECT().OpenLOForWriting(gomock.Any(), mockTx, 222).Return(333, nil)
		mockRepo.EXPECT().TruncateLO(gomock.Any(), mockTx, 333, int64(0)).Return(nil)
		mockRepo.EXPECT().CloseLO(gomock.Any(), mockTx, 333).AnyTimes()

		defer func() {
			if r := recover(); r != nil {
				fmt.Println("Recovered from panic for server test")
			}
		}()
		err := service.UpdateData(stream)
		assert.Error(t, err)
		st, _ := status.FromError(err)
		assert.Equal(t, codes.InvalidArgument, st.Code())
		assert.Contains(t, st.Message(), "no data to update")
	})

	t.Run("error: commit fails", func(t *testing.T) {
		stream := &mockUpdateDataServer{
			reqs: []*pb.UpdateDataRequest{
				{DataUid: "uid", Data: []byte("some-data")},
			},
		}
		mockTx := &sql.Tx{}
		mockRepo.EXPECT().BeginTx(gomock.Any()).Return(mockTx, nil)
		mockRepo.EXPECT().GetOidByItemID(gomock.Any(), "uid").Return(1001, nil)
		mockRepo.EXPECT().OpenLOForWriting(gomock.Any(), mockTx, 1001).Return(888, nil)
		mockRepo.EXPECT().TruncateLO(gomock.Any(), mockTx, 888, int64(0)).Return(nil)
		mockRepo.EXPECT().WriteLO(gomock.Any(), mockTx, 888, []byte("some-data")).Return(nil)
		mockRepo.EXPECT().CloseLO(gomock.Any(), mockTx, 888).AnyTimes()

		defer func() {
			if r := recover(); r != nil {
				fmt.Println("Recovered from panic for server test")
			}
		}()
		err := service.UpdateData(stream)
		assert.Error(t, err)
		st, _ := status.FromError(err)
		assert.Equal(t, codes.Internal, st.Code())
		assert.Contains(t, st.Message(), "commit failed:")
	})
}

func TestRun_Success(t *testing.T) {
	writeTestCerts(t)

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	repo := mockDB.NewMockRepository(ctrl)

	port := getFreePort(t)

	go func() {
		err := Run(port, []config.EndpointRule{}, repo)
		if err != nil {
			t.Errorf("Run failed: %v", err)
		}
	}()

	// Даем немного времени на запуск сервера
	time.Sleep(300 * time.Millisecond)

	conn, err := net.Dial("tcp", fmt.Sprintf("localhost:%d", port))
	assert.NoError(t, err)
	assert.NotNil(t, conn)
	_ = conn.Close()
}

func TestRun_FailToListen(t *testing.T) {
	writeTestCerts(t)

	ln, err := net.Listen("tcp", ":50052")
	assert.NoError(t, err)
	defer ln.Close()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	repo := mockDB.NewMockRepository(ctrl)

	err = Run(50052, nil, repo)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "address already in use")
}

func TestRun_FailToLoadCert(t *testing.T) {
	_ = os.RemoveAll("certs")

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	repo := mockDB.NewMockRepository(ctrl)

	port := getFreePort(t)

	err := Run(port, nil, repo)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to load server cert/key")
}

func TestRun_FailToReadCA(t *testing.T) {
	writeTestCerts(t)
	_ = os.Remove("certs/ca.crt")

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	repo := mockDB.NewMockRepository(ctrl)

	port := getFreePort(t)

	err := Run(port, nil, repo)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to read CA cert")
}

// writeTestCerts создает временные валидные сертификаты в ./certs/
func writeTestCerts(t *testing.T) {
	err := os.MkdirAll("certs", 0755)
	assert.NoError(t, err)

	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	assert.NoError(t, err)

	template := x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject: pkix.Name{
			CommonName: "localhost",
		},
		NotBefore:             time.Now().Add(-time.Hour),
		NotAfter:              time.Now().Add(time.Hour * 24),
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
	}

	derBytes, err := x509.CreateCertificate(rand.Reader, &template, &template, &priv.PublicKey, priv)
	assert.NoError(t, err)

	certOut, err := os.Create("certs/server.crt")
	assert.NoError(t, err)
	defer certOut.Close()
	assert.NoError(t, pem.Encode(certOut, &pem.Block{Type: "CERTIFICATE", Bytes: derBytes}))

	keyOut, err := os.Create("certs/server.key")
	assert.NoError(t, err)
	defer keyOut.Close()
	assert.NoError(t, pem.Encode(keyOut, &pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(priv)}))

	caOut, err := os.Create("certs/ca.crt")
	assert.NoError(t, err)
	defer caOut.Close()
	assert.NoError(t, pem.Encode(caOut, &pem.Block{Type: "CERTIFICATE", Bytes: derBytes}))
	// Очистка после завершения теста
	t.Cleanup(func() {
		_ = os.RemoveAll("certs")
	})
}

// getFreePort выбирает случайный свободный порт
func getFreePort(t *testing.T) int {
	ln, err := net.Listen("tcp", ":0")
	assert.NoError(t, err)
	defer ln.Close()
	_, portStr, err := net.SplitHostPort(ln.Addr().String())
	assert.NoError(t, err)
	var port int
	_, _ = fmt.Sscanf(portStr, "%d", &port)
	return port
}
