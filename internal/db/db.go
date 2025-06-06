package db

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	pb "github.com/fngoc/gault/gen/go/api/proto/v1"
	sqlc "github.com/fngoc/gault/gen/go/db"
	"github.com/fngoc/gault/pkg/logger"
	"github.com/fngoc/gault/pkg/utils"

	"github.com/google/uuid"
	"github.com/pressly/goose"

	_ "github.com/lib/pq"
	"golang.org/x/crypto/bcrypt"
)

// Store структура для работы с хранилищем данных
type Store struct {
	db *sql.DB
}

// InitializePostgresDB инициализация базы данных
func InitializePostgresDB(dbConf string) (Repository, error) {
	postgresInstant, err := sql.Open("postgres", dbConf)
	if err != nil {
		return nil, fmt.Errorf("failed to open database connection: %w", err)
	}

	if err := runMigrations(postgresInstant); err != nil {
		return nil, fmt.Errorf("failed to run migrations: %w", err)
	}

	logger.LogInfo("connected to postgres database")
	return &Store{db: postgresInstant}, nil
}

func runMigrations(db *sql.DB) error {
	if err := goose.SetDialect("postgres"); err != nil {
		return err
	}
	return goose.Up(db, "db/migrations")
}

// GetData получение данных
func (s *Store) GetData(ctx context.Context, id string) (*pb.GetDataResponse, error) {
	ctxDB, cancel := context.WithTimeout(ctx, 7*time.Second)
	defer cancel()

	tx, err := s.db.BeginTx(ctxDB, nil)
	if err != nil {
		return nil, fmt.Errorf("begin transaction: %w", err)
	}

	q := sqlc.New(tx)
	info, err := q.GetDataInfoByID(ctxDB, stringToNullUUID(id).UUID)
	if err != nil {
		return nil, fmt.Errorf("query failed: %w", err)
	}

	// Открываем LO
	var fd int
	if err = tx.QueryRowContext(ctxDB, `SELECT lo_open($1, 131072)`, info.LargeobjectOid).Scan(&fd); err != nil {
		return nil, fmt.Errorf("lo_open failed: %w", err)
	}
	defer func() {
		_, _ = tx.ExecContext(ctxDB, `SELECT lo_close($1)`, fd)
	}()

	// Читаем по чанкам
	var result []byte
	const chunkSize = 1024 * 1024
	for {
		var chunk []byte
		err = tx.QueryRowContext(ctxDB, `SELECT loread($1, $2)`, fd, chunkSize).Scan(&chunk)
		if err == sql.ErrNoRows || len(chunk) == 0 {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("loread failed: %w", err)
		}
		result = append(result, chunk...)
	}

	if err = tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit failed: %w", err)
	}

	// Сборка ответа
	if info.DataType == "file" {
		return &pb.GetDataResponse{
			Type: info.DataType,
			Content: &pb.GetDataResponse_FileData{
				FileData: result,
			},
		}, nil
	}
	return &pb.GetDataResponse{
		Type: info.DataType,
		Content: &pb.GetDataResponse_TextData{
			TextData: string(result),
		},
	}, nil
}

// GetDataNameList получение листа информации о данных
func (s *Store) GetDataNameList(ctx context.Context, userUID string) (*pb.GetUserDataListResponse, error) {
	ctxDB, cancel := context.WithTimeout(ctx, 7*time.Second)
	defer cancel()

	q := sqlc.New(s.db)
	rows, err := q.ListUserData(ctxDB, stringToNullUUID(userUID))
	if err != nil {
		return nil, fmt.Errorf("query error: %w", err)
	}

	items := make([]*pb.UserDataItem, 0, len(rows))
	for _, row := range rows {
		items = append(items, &pb.UserDataItem{
			Id:   row.ID.String(),
			Type: row.DataType,
			Name: row.DataName,
		})
	}

	return &pb.GetUserDataListResponse{Items: items}, nil
}

// CreateUser создание пользователя
func (s *Store) CreateUser(ctx context.Context, username, passwordHash string) (string, string, error) {
	ctxDB, cancel := context.WithTimeout(ctx, 7*time.Second)
	defer cancel()

	tx, err := s.db.BeginTx(ctxDB, nil)
	if err != nil {
		return "", "", fmt.Errorf("failed to begin transaction: %w", err)
	}

	q := sqlc.New(tx)
	userID, err := q.CreateUser(ctxDB, sqlc.CreateUserParams{
		Username:     username,
		PasswordHash: passwordHash,
	})
	if err != nil {
		_ = tx.Rollback()
		return "", "", fmt.Errorf("failed to create user: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return "", "", fmt.Errorf("failed to commit transaction: %w", err)
	}

	token, err := s.createSessionToken(ctxDB, userID.String())
	if err != nil {
		return "", "", err
	}
	return userID.String(), token, nil
}

// IsUserCreated проверка на существование пользователя
func (s *Store) IsUserCreated(ctx context.Context, username string) (bool, error) {
	ctxDB, cancel := context.WithTimeout(ctx, 7*time.Second)
	defer cancel()

	q := sqlc.New(s.db)
	isCreated, err := q.IsUserCreated(ctxDB, username)
	if err != nil {
		return false, fmt.Errorf("failed to check if user exists: %w", err)
	}

	return isCreated, nil
}

// CheckSessionUser проверка сессии пользователя
func (s *Store) CheckSessionUser(ctx context.Context, userUID, token string) bool {
	ctxDB, cancel := context.WithTimeout(ctx, 7*time.Second)
	defer cancel()

	q := sqlc.New(s.db)
	isValid, err := q.CheckSessionUser(ctxDB, sqlc.CheckSessionUserParams{
		UserID:       stringToNullUUID(userUID),
		SessionToken: token,
	})
	if err != nil {
		return false
	}
	return isValid
}

// UpdateSessionUser обновление сессии пользователя
func (s *Store) UpdateSessionUser(ctx context.Context, username, password string) (string, string, error) {
	ctxDB, cancel := context.WithTimeout(ctx, 7*time.Second)
	defer cancel()

	q := sqlc.New(s.db)
	user, err := q.GetUserCredentialsByUsername(ctxDB, username)
	if err != nil {
		return "", "", fmt.Errorf("user lookup failed: %w", err)
	}

	// Сравниваем хеш
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		return "", "", fmt.Errorf("invalid password")
	}

	// Создаём токен
	token, err := s.createSessionToken(ctxDB, user.ID.String())
	if err != nil {
		return "", "", fmt.Errorf("failed to create session token: %w", err)
	}

	return user.ID.String(), token, nil
}

// DeleteData удаление данных
func (s *Store) DeleteData(ctx context.Context, id string) error {
	ctxDB, cancel := context.WithTimeout(ctx, 7*time.Second)
	defer cancel()

	tx, err := s.db.BeginTx(ctxDB, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}

	q := sqlc.New(tx)
	if err := q.DeleteUserData(ctxDB, stringToNullUUID(id).UUID); err != nil {
		_ = tx.Rollback()
		return fmt.Errorf("failed to delete user data: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}
	return nil
}

// createSessionToken создание токена для пользователя
func (s *Store) createSessionToken(ctx context.Context, userUID string) (string, error) {
	ctxDB, cancel := context.WithTimeout(ctx, 7*time.Second)
	defer cancel()

	tx, err := s.db.BeginTx(ctxDB, nil)
	if err != nil {
		return "", fmt.Errorf("failed to begin transaction: %w", err)
	}

	q := sqlc.New(tx)
	sessionToken, err := utils.GenerateToken()
	if err != nil {
		return "", fmt.Errorf("failed to generate token: %w", err)
	}

	err = q.InsertUserSession(ctxDB, sqlc.InsertUserSessionParams{
		UserID:       stringToNullUUID(userUID),
		SessionToken: sessionToken,
	})
	if err != nil {
		_ = tx.Rollback()
		return "", fmt.Errorf("failed to insert session: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return "", fmt.Errorf("failed to commit transaction: %w", err)
	}
	return sessionToken, nil
}

// stringToNullUUID перевод строки в UUID
func stringToNullUUID(s string) uuid.NullUUID {
	u, err := uuid.Parse(s)
	if err != nil {
		return uuid.NullUUID{Valid: false}
	}
	return uuid.NullUUID{UUID: u, Valid: true}
}

// BeginTx начало транзакции
func (s *Store) BeginTx(ctx context.Context) (*sql.Tx, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	return tx, nil
}

// CreateEmptyLO создание пустого файла в БД
func (s *Store) CreateEmptyLO(ctx context.Context, tx *sql.Tx) (int, error) {
	var oid int
	if err := tx.QueryRowContext(ctx, `SELECT lo_create(0)`).Scan(&oid); err != nil {
		return 0, fmt.Errorf("lo_create failed: %w", err)
	}
	return oid, nil
}

func (s *Store) GetOidByItemID(ctx context.Context, itemID string) (int, error) {
	q := sqlc.New(s.db)
	oid, err := q.GetOidByID(ctx, stringToNullUUID(itemID).UUID)
	if err != nil {
		return 0, fmt.Errorf("failed to get oid by item id: %w", err)
	}
	return int(oid), nil
}

// InsertUserDataRecordTx вставка данных
func (s *Store) InsertUserDataRecordTx(ctx context.Context, tx *sql.Tx, userDataID, userUID, dataType, dataName string, oid int) error {
	q := sqlc.New(tx)
	err := q.InsertUserDataWithOid(ctx, sqlc.InsertUserDataWithOidParams{
		ID:             stringToNullUUID(userDataID).UUID,
		UserID:         stringToNullUUID(userUID),
		DataType:       dataType,
		DataName:       dataName,
		LargeobjectOid: uint32(oid),
	})
	if err != nil {
		return fmt.Errorf("insert user_data failed: %w", err)
	}
	return nil
}

// OpenLOForWriting открывает LO один раз
func (s *Store) OpenLOForWriting(ctx context.Context, tx *sql.Tx, oid int) (int, error) {
	const invWrite = 131072
	var fd int
	if err := tx.QueryRowContext(ctx, `SELECT lo_open($1, $2)`, oid, invWrite).Scan(&fd); err != nil {
		return 0, fmt.Errorf("lo_open failed: %w", err)
	}
	return fd, nil
}

// WriteLO записывает чанк в открытый LO
func (s *Store) WriteLO(ctx context.Context, tx *sql.Tx, fd int, chunk []byte) error {
	var wrote int
	if err := tx.QueryRowContext(ctx, `SELECT lowrite($1, $2)`, fd, chunk).Scan(&wrote); err != nil {
		return fmt.Errorf("lowrite failed: %w", err)
	}
	if wrote != len(chunk) {
		return fmt.Errorf("partial write: expected %d, wrote %d", len(chunk), wrote)
	}
	return nil
}

// CloseLO закрывает файловый дескриптор LO
func (s *Store) CloseLO(ctx context.Context, tx *sql.Tx, fd int) {
	_, _ = tx.ExecContext(ctx, `SELECT lo_close($1)`, fd)
}

// TruncateLO обнуляет содержимое LO, делая его длину равной newSize
func (s *Store) TruncateLO(ctx context.Context, tx *sql.Tx, fd int, newSize int64) error {
	_, err := tx.ExecContext(ctx, `SELECT lo_truncate($1, $2)`, fd, newSize)
	if err != nil {
		return fmt.Errorf("lo_truncate failed: %w", err)
	}
	return nil
}
