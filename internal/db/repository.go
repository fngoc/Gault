package db

import (
	pb "Gault/gen/go/api/proto/v1"
	"context"
	"database/sql"
)

// Repository интерфейс взаимодействия с хранилищем
type Repository interface {
	GetData(context.Context, string) (*pb.GetDataResponse, error)
	GetDataNameList(context.Context, string) (*pb.GetUserDataListResponse, error)
	GetOidByItemID(context.Context, string) (int, error)
	CreateUser(context.Context, string, string) (string, string, error)
	IsUserCreated(context.Context, string) (bool, error)
	CheckSessionUser(context.Context, string, string) bool
	UpdateSessionUser(context.Context, string, string) (string, string, error)
	DeleteData(context.Context, string) error

	BeginTx(context.Context) (*sql.Tx, error)
	CreateEmptyLO(context.Context, *sql.Tx) (int, error)
	InsertUserDataRecordTx(context.Context, *sql.Tx, string, string, string, string, int) error

	OpenLOForWriting(ctx context.Context, tx *sql.Tx, oid int) (int, error)
	WriteLO(ctx context.Context, tx *sql.Tx, fd int, chunk []byte) error
	CloseLO(ctx context.Context, tx *sql.Tx, fd int)
	TruncateLO(context.Context, *sql.Tx, int, int64) error
}
