package db

import (
	"context"
	"database/sql"
	"testing"

	"golang.org/x/crypto/bcrypt"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
)

func setupMockDB(t *testing.T) (*sql.DB, sqlmock.Sqlmock, Repository) {
	dbMock, mock, err := sqlmock.New()
	assert.NoError(t, err)
	store := &Store{db: dbMock}
	return dbMock, mock, store
}

func TestInitializePostgresDB(t *testing.T) {
	dbMock, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer dbMock.Close()

	mock.ExpectExec(`(?i)CREATE TABLE IF NOT EXISTS users`).WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectExec(`(?i)CREATE TABLE IF NOT EXISTS user_data`).WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectExec(`(?i)CREATE TABLE IF NOT EXISTS user_sessions`).WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectPing()

	_, err = InitializePostgresDB("mock-dsn")
	assert.Error(t, err)
}

func TestSaveData(t *testing.T) {
	dbMock, mock, store := setupMockDB(t)
	defer dbMock.Close()

	ctx := context.Background()
	mock.ExpectExec("INSERT INTO user_data").
		WithArgs("user-uid", "type", "name", []byte("data")).
		WillReturnResult(sqlmock.NewResult(1, 1))

	err := store.SaveData(ctx, "user-uid", "type", "name", []byte("data"))
	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetData(t *testing.T) {
	dbMock, mock, store := setupMockDB(t)
	defer dbMock.Close()

	ctx := context.Background()
	mock.ExpectQuery(`(?i)SELECT\s+data_type,\s+data_encrypted\s+FROM\s+user_data\s+WHERE\s+id\s*=\s*\$?`).
		WithArgs("data-id").
		WillReturnRows(sqlmock.NewRows([]string{"data_type", "data_encrypted"}).
			AddRow("text", []byte("sample data")))

	resp, err := store.GetData(ctx, "data-id")
	assert.NoError(t, err)
	assert.Equal(t, "text", resp.Type)
	assert.Equal(t, "sample data", resp.GetTextData())
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestCreateUser(t *testing.T) {
	dbMock, mock, store := setupMockDB(t)
	defer dbMock.Close()

	ctx := context.Background()
	mock.ExpectQuery(`(?i)INSERT\s+INTO\s+users\s*\(username,\s*password_hash\)\s*VALUES\s*\(\$1,\s*\$2\)\s*RETURNING\s*id`).
		WithArgs("testuser", "hashed-password").
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow("user-uid"))

	mock.ExpectExec(`(?i)INSERT\s+INTO\s+user_sessions\s*\(user_id, session_token, expires_at\)`).
		WithArgs("user-uid", sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(1, 1))

	uid, token, err := store.CreateUser(ctx, "testuser", "hashed-password")
	assert.NoError(t, err)
	assert.Equal(t, "user-uid", uid)
	assert.NotEmpty(t, token)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestIsUserCreated(t *testing.T) {
	dbMock, mock, store := setupMockDB(t)
	defer dbMock.Close()

	ctx := context.Background()
	mock.ExpectQuery(`(?i)SELECT\s+EXISTS\s*\(SELECT\s+1\s+FROM\s+users\s+WHERE\s+username\s*=\s*\$1\)`).
		WithArgs("testuser").
		WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))

	exists, err := store.IsUserCreated(ctx, "testuser")
	assert.NoError(t, err)
	assert.True(t, exists)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestDeleteData(t *testing.T) {
	dbMock, mock, store := setupMockDB(t)
	defer dbMock.Close()

	ctx := context.Background()
	mock.ExpectExec(`(?i)DELETE\s+FROM\s+user_data\s+WHERE\s+id\s*=\s*\$1`).
		WithArgs("data-id").
		WillReturnResult(sqlmock.NewResult(1, 1))

	err := store.DeleteData(ctx, "data-id")
	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUpdateData(t *testing.T) {
	dbMock, mock, store := setupMockDB(t)
	defer dbMock.Close()

	ctx := context.Background()
	mock.ExpectExec(`(?i)UPDATE\s+user_data\s+SET\s+data_encrypted\s*=\s*\$1\s+WHERE\s+id\s*=\s*\$2`).
		WithArgs([]byte("new data"), "data-id").
		WillReturnResult(sqlmock.NewResult(1, 1))

	err := store.UpdateData(ctx, "data-id", []byte("new data"))
	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUpdateSessionUser(t *testing.T) {
	dbMock, mock, store := setupMockDB(t)
	defer dbMock.Close()

	ctx := context.Background()

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte("password"), bcrypt.DefaultCost)
	assert.NoError(t, err) // Проверяем, что хеширование прошло успешно

	mock.ExpectQuery(`(?i)SELECT\s+id,\s+password_hash\s+FROM\s+users\s+WHERE\s+username\s*=\s*\$1`).
		WithArgs("testuser").
		WillReturnRows(sqlmock.NewRows([]string{"id", "password_hash"}).
			AddRow("user-uid", string(hashedPassword)))

	mock.ExpectExec(`(?i)INSERT\s+INTO\s+user_sessions\s*\(user_id, session_token, expires_at\)`).
		WithArgs("user-uid", sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(1, 1))

	uid, token, err := store.UpdateSessionUser(ctx, "testuser", "password")
	assert.NoError(t, err)
	assert.Equal(t, "user-uid", uid)
	assert.NotEmpty(t, token)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestCheckSessionUser(t *testing.T) {
	dbMock, mock, store := setupMockDB(t)
	defer dbMock.Close()

	ctx := context.Background()

	mock.ExpectQuery(`(?i)SELECT\s+EXISTS\s*\(SELECT\s+1\s+FROM\s+user_sessions\s+WHERE\s+user_id\s*=\s*\$1\s+AND\s+session_token\s*=\s*\$2\)`).
		WithArgs("user-uid", "valid-token").
		WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))

	isValid := store.CheckSessionUser(ctx, "user-uid", "valid-token")
	assert.True(t, isValid)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetDataNameList(t *testing.T) {
	dbMock, mock, store := setupMockDB(t)
	defer dbMock.Close()

	ctx := context.Background()

	mock.ExpectQuery(`(?i)SELECT\s+id,\s+data_type,\s+data_name\s+FROM\s+user_data\s+WHERE\s+user_id\s*=\s*\$1`).
		WithArgs("user-uid").
		WillReturnRows(sqlmock.NewRows([]string{"id", "data_type", "data_name"}).
			AddRow("data-id-1", "type1", "name1").
			AddRow("data-id-2", "type2", "name2"))

	resp, err := store.GetDataNameList(ctx, "user-uid")
	assert.NoError(t, err)
	assert.Len(t, resp.Items, 2)
	assert.Equal(t, "data-id-1", resp.Items[0].Id)
	assert.Equal(t, "type1", resp.Items[0].Type)
	assert.Equal(t, "name1", resp.Items[0].Name)
	assert.Equal(t, "data-id-2", resp.Items[1].Id)
	assert.Equal(t, "type2", resp.Items[1].Type)
	assert.Equal(t, "name2", resp.Items[1].Name)
	assert.NoError(t, mock.ExpectationsWereMet())
}
