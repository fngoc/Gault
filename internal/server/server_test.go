package server

import (
	"context"
	"fmt"
	"net"
	"testing"

	"google.golang.org/grpc/metadata"

	pb "Gault/gen/go/api/proto/v1"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"

	mockDB "Gault/gen/go/db"
)

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

func TestGaultService_SaveData(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	repo := mockDB.NewMockRepository(ctrl)
	service := &GaultService{rep: repo}

	t.Run("success", func(t *testing.T) {
		md := metadata.New(map[string]string{"useruid": "user-uid"})
		ctx := metadata.NewIncomingContext(context.Background(), md)
		repo.EXPECT().SaveData(ctx, "user-uid", "type", "name", []byte("data")).Return(nil)

		resp, err := service.SaveData(ctx, &pb.SaveDataRequest{UserUid: "user-uid", Type: "type", Name: "name", Data: []byte("data")})
		assert.NoError(t, err)
		assert.NotNil(t, resp)
	})
	t.Run("error", func(t *testing.T) {
		md := metadata.New(map[string]string{"useruid": "user-uid"})
		ctx := metadata.NewIncomingContext(context.Background(), md)
		repo.EXPECT().SaveData(ctx, "user-uid", "type", "name", []byte("data")).Return(fmt.Errorf("error"))

		resp, err := service.SaveData(ctx, &pb.SaveDataRequest{UserUid: "user-uid", Type: "type", Name: "name", Data: []byte("data")})
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

func TestGaultService_UpdateData(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	repo := mockDB.NewMockRepository(ctrl)
	service := &GaultService{rep: repo}

	t.Run("success", func(t *testing.T) {
		md := metadata.New(map[string]string{"useruid": "user-uid"})
		ctx := metadata.NewIncomingContext(context.Background(), md)
		repo.EXPECT().UpdateData(ctx, "data-id", []byte("new-data")).Return(nil)

		resp, err := service.UpdateData(ctx, &pb.UpdateDataRequest{Id: "data-id", Data: []byte("new-data")})
		assert.NoError(t, err)
		assert.NotNil(t, resp)
	})
	t.Run("success", func(t *testing.T) {
		md := metadata.New(map[string]string{"useruid": "user-uid"})
		ctx := metadata.NewIncomingContext(context.Background(), md)
		repo.EXPECT().UpdateData(ctx, "data-id", []byte("new-data")).Return(fmt.Errorf("error"))

		resp, err := service.UpdateData(ctx, &pb.UpdateDataRequest{Id: "data-id", Data: []byte("new-data")})
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
		_ = Run(port, repo)
	}()
}
