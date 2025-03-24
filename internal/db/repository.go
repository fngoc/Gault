package db

import (
	pb "Gault/api/pb/api/proto"
	"context"
)

type Repository interface {
	SaveData(context.Context, string, string, string, []byte) error
	GetData(context.Context, string) (*pb.GetDataResponse, error)
	GetDataNameList(context.Context, string) (*pb.GetUserDataListResponse, error)
	CreateUser(context.Context, string, string) (string, string, error)
	IsUserCreated(context.Context, string) (bool, error)
	CheckSessionUser(context.Context, string, string) bool
	UpdateSessionUser(context.Context, string, string) (string, string, error)
	DeleteData(context.Context, string) error
}
