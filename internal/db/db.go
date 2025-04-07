package db

import (
	pb "Gault/gen/go/api/proto/v1"
	sqlc "Gault/gen/go/db"
	"Gault/pkg/logger"
	"Gault/pkg/utils"
	"context"
	"database/sql"
	"fmt"
	"time"

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

// SaveData сохранение данных
func (s *Store) SaveData(ctx context.Context, userUID, dataType, dataName string, data []byte) error {
	ctxDB, cancel := context.WithTimeout(ctx, 7*time.Second)
	defer cancel()

	tx, err := s.db.BeginTx(ctxDB, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}

	q := sqlc.New(tx)
	params := sqlc.SaveDataParams{
		UserID:        stringToNullUUID(userUID),
		DataType:      dataType,
		DataName:      dataName,
		DataEncrypted: data,
	}

	if err := q.SaveData(ctxDB, params); err != nil {
		_ = tx.Rollback()
		return fmt.Errorf("failed to save data: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// GetData получение данных
func (s *Store) GetData(ctx context.Context, id string) (*pb.GetDataResponse, error) {
	ctxDB, cancel := context.WithTimeout(ctx, 7*time.Second)
	defer cancel()

	q := sqlc.New(s.db)
	result, err := q.GetData(ctxDB, stringToNullUUID(id).UUID)
	if err != nil {
		return nil, fmt.Errorf("get user data: %w", err)
	}

	if result.DataType == "file" {
		return &pb.GetDataResponse{
			Type: result.DataType,
			Content: &pb.GetDataResponse_FileData{
				FileData: result.DataEncrypted,
			},
		}, nil
	}
	return &pb.GetDataResponse{
		Type: result.DataType,
		Content: &pb.GetDataResponse_TextData{
			TextData: string(result.DataEncrypted),
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

// UpdateData обновление данных
func (s *Store) UpdateData(ctx context.Context, id string, data []byte) error {
	ctxDB, cancel := context.WithTimeout(ctx, 7*time.Second)
	defer cancel()

	tx, err := s.db.BeginTx(ctxDB, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}

	q := sqlc.New(tx)
	err = q.UpdateUserData(ctxDB, sqlc.UpdateUserDataParams{
		DataEncrypted: data,
		ID:            stringToNullUUID(id).UUID,
	})
	if err != nil {
		_ = tx.Rollback()
		return fmt.Errorf("update failed: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit failed: %w", err)
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

func stringToNullUUID(s string) uuid.NullUUID {
	u, err := uuid.Parse(s)
	if err != nil {
		return uuid.NullUUID{Valid: false}
	}
	return uuid.NullUUID{UUID: u, Valid: true}
}
