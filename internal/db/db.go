package db

import (
	pb "Gault/gen/go/api/proto/v1"
	"Gault/pkg/logger"
	"Gault/pkg/utils"
	"context"
	"database/sql"
	"fmt"
	"time"

	_ "github.com/lib/pq"
	"golang.org/x/crypto/bcrypt"
)

// Store структура для работы с хранилищем данных
type Store struct {
	db *sql.DB
}

// InitializePostgresDB инициализация базы данных
func InitializePostgresDB(dbConf string) (Repository, error) {
	db, err := sql.Open("postgres", dbConf)
	if err != nil {
		return nil, fmt.Errorf("failed to open database connection: %w", err)
	}

	if err := createTables(db); err != nil {
		return nil, fmt.Errorf("failed to create tables: %w", err)
	}

	err = db.Ping()
	if err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}
	logger.Log.Info("connected to postgres database")
	return &Store{db: db}, nil
}

// createTables создание таблиц при запуске
func createTables(db *sql.DB) error {
	query := `
	CREATE TABLE IF NOT EXISTS users(
		id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		username VARCHAR(50) UNIQUE NOT NULL,
		password_hash TEXT NOT NULL,
		created_at TIMESTAMPTZ DEFAULT NOW(),
		updated_at TIMESTAMPTZ DEFAULT NOW()
	);
	CREATE TABLE IF NOT EXISTS user_data(
		id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		user_id UUID REFERENCES users(id) ON DELETE CASCADE,
		data_type VARCHAR(50) NOT NULL,
		data_name VARCHAR(50) NOT NULL,
		data_encrypted BYTEA NOT NULL,
		created_at TIMESTAMPTZ DEFAULT NOW()
	);
	CREATE TABLE IF NOT EXISTS user_sessions(
		id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		user_id UUID REFERENCES users(id) ON DELETE CASCADE,
		session_token TEXT UNIQUE NOT NULL,
		expires_at TIMESTAMPTZ NOT NULL,
		created_at TIMESTAMPTZ DEFAULT NOW()
	);`

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	if _, err := db.ExecContext(ctx, query); err != nil {
		return fmt.Errorf("failed to execute query: %w", err)
	}

	logger.Log.Info("database table created")
	return nil
}

// SaveData сохранение данных
func (s *Store) SaveData(ctx context.Context, userUID, dataType, dataName string, data []byte) error {
	ctxDB, cancel := context.WithTimeout(ctx, 7*time.Second)
	defer cancel()

	query := `INSERT INTO user_data (user_id, data_type, data_name, data_encrypted) VALUES ($1, $2, $3, $4)`
	_, err := s.db.ExecContext(ctxDB, query, userUID, dataType, dataName, data)
	if err != nil {
		return fmt.Errorf("failed to execute query: %w", err)
	}
	return nil
}

// GetData получение данных
func (s *Store) GetData(ctx context.Context, id string) (*pb.GetDataResponse, error) {
	ctxDB, cancel := context.WithTimeout(ctx, 7*time.Second)
	defer cancel()

	var data []byte
	var dataType string
	query := `SELECT data_type, data_encrypted FROM user_data WHERE id = $1`
	err := s.db.QueryRowContext(ctxDB, query, id).Scan(&dataType, &data)
	if err != nil {
		return nil, err
	}

	if dataType == "file" {
		return &pb.GetDataResponse{
			Type: dataType,
			Content: &pb.GetDataResponse_FileData{
				FileData: data,
			},
		}, nil
	}
	return &pb.GetDataResponse{
		Type: dataType,
		Content: &pb.GetDataResponse_TextData{
			TextData: string(data),
		},
	}, nil
}

// GetDataNameList получение листа информации о данных
func (s *Store) GetDataNameList(ctx context.Context, userUID string) (*pb.GetUserDataListResponse, error) {
	ctxDB, cancel := context.WithTimeout(ctx, 7*time.Second)
	defer cancel()

	query := `SELECT id, data_type, data_name FROM user_data WHERE user_id = $1`
	rows, err := s.db.QueryContext(ctxDB, query, userUID)
	if err != nil {
		return nil, fmt.Errorf("failed to execute query: %w", err)
	}
	if rows.Err() != nil {
		return nil, fmt.Errorf("failed to execute query: %w", rows.Err())
	}
	defer rows.Close()

	var list []*pb.UserDataItem
	for rows.Next() {
		var item pb.UserDataItem
		if err := rows.Scan(&item.Id, &item.Type, &item.Name); err != nil {
			return nil, err
		}
		list = append(list, &item)
	}
	return &pb.GetUserDataListResponse{Items: list}, nil
}

// CreateUser создание пользователя
func (s *Store) CreateUser(ctx context.Context, username, passwordHash string) (string, string, error) {
	ctxDB, cancel := context.WithTimeout(ctx, 7*time.Second)
	defer cancel()

	var userUID string
	query := `INSERT INTO users (username, password_hash) VALUES ($1, $2) RETURNING id;`
	err := s.db.QueryRowContext(ctxDB, query, username, passwordHash).Scan(&userUID)
	if err != nil {
		return "", "", err
	}

	token, err := s.createSessionToken(ctxDB, userUID)
	if err != nil {
		return "", "", err
	}
	return userUID, token, nil
}

// IsUserCreated проверка на существование пользователя
func (s *Store) IsUserCreated(ctx context.Context, username string) (bool, error) {
	ctxDB, cancel := context.WithTimeout(ctx, 7*time.Second)
	defer cancel()

	query := `SELECT EXISTS (SELECT 1 FROM users WHERE username = $1)`
	row := s.db.QueryRowContext(ctxDB, query, username)
	if row.Err() != nil {
		return false, row.Err()
	}

	var isCreated bool
	err := row.Scan(&isCreated)
	if err != nil {
		return false, fmt.Errorf("failed to check if user exists: %w", err)
	}
	return isCreated, nil
}

// CheckSessionUser проверка сессии пользователя
func (s *Store) CheckSessionUser(ctx context.Context, userUID, token string) bool {
	ctxDB, cancel := context.WithTimeout(ctx, 7*time.Second)
	defer cancel()

	var isCreated bool
	query := `SELECT EXISTS (SELECT 1 FROM user_sessions WHERE user_id = $1 AND session_token = $2)`
	err := s.db.QueryRowContext(ctxDB, query, userUID, token).Scan(&isCreated)
	if err != nil {
		return false
	}
	return isCreated
}

// UpdateSessionUser обновление сессии пользователя
func (s *Store) UpdateSessionUser(ctx context.Context, username, password string) (string, string, error) {
	ctxDB, cancel := context.WithTimeout(ctx, 7*time.Second)
	defer cancel()

	var userUID string
	var hashedPassword string
	query := `SELECT id, password_hash FROM users WHERE username = $1`
	err := s.db.QueryRowContext(ctxDB, query, username).Scan(&userUID, &hashedPassword)
	if err != nil {
		return "", "", err
	}
	err = bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(password))
	if err != nil {
		return "", "", err
	}

	token, err := s.createSessionToken(ctxDB, userUID)
	if err != nil {
		return "", "", err
	}
	return userUID, token, nil
}

// DeleteData удаление данных
func (s *Store) DeleteData(ctx context.Context, id string) error {
	ctxDB, cancel := context.WithTimeout(ctx, 7*time.Second)
	defer cancel()

	query := `DELETE FROM user_data WHERE id = $1`
	_, err := s.db.ExecContext(ctxDB, query, id)
	return err
}

func (s *Store) UpdateData(ctx context.Context, id string, data []byte) error {
	ctxDB, cancel := context.WithTimeout(ctx, 7*time.Second)
	defer cancel()

	query := `UPDATE user_data SET data_encrypted = $1 WHERE id = $2`
	_, err := s.db.ExecContext(ctxDB, query, data, id)
	return err
}

// createSessionToken создание токена для пользователя
func (s *Store) createSessionToken(ctx context.Context, userUID string) (string, error) {
	ctxDB, cancel := context.WithTimeout(ctx, 7*time.Second)
	defer cancel()

	sessionToke, err := utils.GenerateToken()
	if err != nil {
		return "", err
	}

	query := `INSERT INTO user_sessions (user_id, session_token, expires_at) VALUES ($1, $2, NOW() + INTERVAL '20 minutes')`
	_, err = s.db.ExecContext(ctxDB, query, userUID, sessionToke)
	if err != nil {
		return "", err
	}
	return sessionToke, nil
}
