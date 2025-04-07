package db

import (
	"context"
	"database/sql"
	"errors"
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

	mock.ExpectBegin()
	ctx := context.Background()
	mock.ExpectExec("INSERT INTO user_data").
		WithArgs("3a0a4950-16e3-4720-814b-17e6b4fd0bc2", "type", "name", []byte("data")).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	err := store.SaveData(ctx, "3a0a4950-16e3-4720-814b-17e6b4fd0bc2", "type", "name", []byte("data"))
	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestSaveDataError(t *testing.T) {
	dbMock, mock, store := setupMockDB(t)
	defer dbMock.Close()

	mock.ExpectBegin()
	ctx := context.Background()
	mock.ExpectExec("INSERT INTO user_data").
		WithArgs("user-uid", "type", "name", []byte("data")).
		WillReturnError(errors.New("error"))
	mock.ExpectRollback()

	err := store.SaveData(ctx, "user-uid", "type", "name", []byte("data"))
	assert.Error(t, err)
}

func TestGetDataText(t *testing.T) {
	dbMock, mock, store := setupMockDB(t)
	defer dbMock.Close()

	ctx := context.Background()
	mock.ExpectQuery(`(?i)SELECT\s+data_type,\s+data_encrypted\s+FROM\s+user_data\s+WHERE\s+id\s*=\s*\$?`).
		WithArgs("3a0a4950-16e3-4720-814b-17e6b4fd0bc2").
		WillReturnRows(sqlmock.NewRows([]string{"data_type", "data_encrypted"}).
			AddRow("text", []byte("sample data")))

	resp, err := store.GetData(ctx, "3a0a4950-16e3-4720-814b-17e6b4fd0bc2")
	assert.NoError(t, err)
	assert.Equal(t, "text", resp.Type)
	assert.Equal(t, "sample data", resp.GetTextData())
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetDataFile(t *testing.T) {
	dbMock, mock, store := setupMockDB(t)
	defer dbMock.Close()

	ctx := context.Background()
	mock.ExpectQuery(`(?i)SELECT\s+data_type,\s+data_encrypted\s+FROM\s+user_data\s+WHERE\s+id\s*=\s*\$?`).
		WithArgs("3a0a4950-16e3-4720-814b-17e6b4fd0bc2").
		WillReturnRows(sqlmock.NewRows([]string{"data_type", "data_encrypted"}).
			AddRow("file", []byte("sample data")))

	resp, err := store.GetData(ctx, "3a0a4950-16e3-4720-814b-17e6b4fd0bc2")
	assert.NoError(t, err)
	assert.Equal(t, "file", resp.Type)
	assert.Equal(t, "sample data", string(resp.GetFileData()))
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestCreateUser(t *testing.T) {
	dbMock, mock, store := setupMockDB(t)
	defer dbMock.Close()

	mock.ExpectBegin()
	ctx := context.Background()
	mock.ExpectQuery(`(?i)INSERT\s+INTO\s+users\s*\(username,\s*password_hash\)\s*VALUES\s*\(\$1,\s*\$2\)\s*RETURNING\s*id`).
		WithArgs("testuser", "hashed-password").
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow("3a0a4950-16e3-4720-814b-17e6b4fd0bc2"))
	mock.ExpectCommit()

	mock.ExpectBegin()
	mock.ExpectExec(`(?i)INSERT\s+INTO\s+user_sessions\s*\(user_id, session_token, expires_at\)`).
		WithArgs("3a0a4950-16e3-4720-814b-17e6b4fd0bc2", sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	uid, token, err := store.CreateUser(ctx, "testuser", "hashed-password")
	assert.NoError(t, err)
	assert.Equal(t, "3a0a4950-16e3-4720-814b-17e6b4fd0bc2", uid)
	assert.NotEmpty(t, token)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestCreateUserError(t *testing.T) {
	dbMock, mock, store := setupMockDB(t)
	defer dbMock.Close()

	mock.ExpectBegin()
	ctx := context.Background()
	mock.ExpectQuery(`(?i)INSERT\s+INTO\s+users\s*\(username,\s*password_hash\)\s*VALUES\s*\(\$1,\s*\$2\)\s*RETURNING\s*id`).
		WithArgs("testuser", "hashed-password").
		WillReturnError(errors.New("error"))
	mock.ExpectRollback()

	_, _, err := store.CreateUser(ctx, "testuser", "hashed-password")
	assert.Error(t, err)
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

func TestIsUserCreatedError(t *testing.T) {
	dbMock, mock, store := setupMockDB(t)
	defer dbMock.Close()

	ctx := context.Background()
	mock.ExpectQuery(`(?i)SELECT\s+EXISTS\s*\(SELECT\s+1\s+FROM\s+users\s+WHERE\s+username\s*=\s*\$1\)`).
		WithArgs("testuser").
		WillReturnError(errors.New("error"))

	exists, err := store.IsUserCreated(ctx, "testuser")
	assert.Error(t, err)
	assert.False(t, exists)
}

func TestDeleteData(t *testing.T) {
	dbMock, mock, store := setupMockDB(t)
	defer dbMock.Close()

	mock.ExpectBegin()
	ctx := context.Background()
	mock.ExpectExec(`(?i)DELETE\s+FROM\s+user_data\s+WHERE\s+id\s*=\s*\$1`).
		WithArgs("3a0a4950-16e3-4720-814b-17e6b4fd0bc2").
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	err := store.DeleteData(ctx, "3a0a4950-16e3-4720-814b-17e6b4fd0bc2")
	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestDeleteDataError(t *testing.T) {
	dbMock, mock, store := setupMockDB(t)
	defer dbMock.Close()

	mock.ExpectBegin()
	ctx := context.Background()
	mock.ExpectExec(`(?i)DELETE\s+FROM\s+user_data\s+WHERE\s+id\s*=\s*\$1`).
		WithArgs("data-id").
		WillReturnError(errors.New("error"))
	mock.ExpectRollback()

	err := store.DeleteData(ctx, "data-id")
	assert.Error(t, err)
}

func TestUpdateData(t *testing.T) {
	dbMock, mock, store := setupMockDB(t)
	defer dbMock.Close()

	mock.ExpectBegin()
	ctx := context.Background()
	mock.ExpectExec(`(?i)UPDATE\s+user_data\s+SET\s+data_encrypted\s*=\s*\$1\s+WHERE\s+id\s*=\s*\$2`).
		WithArgs([]byte("new data"), "3a0a4950-16e3-4720-814b-17e6b4fd0bc2").
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	err := store.UpdateData(ctx, "3a0a4950-16e3-4720-814b-17e6b4fd0bc2", []byte("new data"))
	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUpdateDataError(t *testing.T) {
	dbMock, mock, store := setupMockDB(t)
	defer dbMock.Close()

	mock.ExpectBegin()
	ctx := context.Background()
	mock.ExpectExec(`(?i)UPDATE\s+user_data\s+SET\s+data_encrypted\s*=\s*\$1\s+WHERE\s+id\s*=\s*\$2`).
		WithArgs([]byte("new data"), "3a0a4950-16e3-4720-814b-17e6b4fd0bc2").
		WillReturnError(errors.New("error"))
	mock.ExpectRollback()

	err := store.UpdateData(ctx, "3a0a4950-16e3-4720-814b-17e6b4fd0bc2", []byte("new data"))
	assert.Error(t, err)
}

func TestUpdateSessionUser(t *testing.T) {
	dbMock, mock, store := setupMockDB(t)
	defer dbMock.Close()

	ctx := context.Background()

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte("password"), bcrypt.DefaultCost)
	assert.NoError(t, err)

	mock.ExpectQuery(`(?i)SELECT\s+id,\s+password_hash\s+FROM\s+users\s+WHERE\s+username\s*=\s*\$1`).
		WithArgs("testuser").
		WillReturnRows(sqlmock.NewRows([]string{"3a0a4950-16e3-4720-814b-17e6b4fd0bc5", "password_hash"}).
			AddRow("3a0a4950-16e3-4720-814b-17e6b4fd0bc4", string(hashedPassword)))

	mock.ExpectBegin()
	mock.ExpectExec(`(?i)INSERT\s+INTO\s+user_sessions\s*\(user_id, session_token, expires_at\)`).
		WithArgs("3a0a4950-16e3-4720-814b-17e6b4fd0bc4", sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	uid, token, err := store.UpdateSessionUser(ctx, "testuser", "password")
	assert.NoError(t, err)
	assert.Equal(t, "3a0a4950-16e3-4720-814b-17e6b4fd0bc4", uid)
	assert.NotEmpty(t, token)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUpdateSessionUserError(t *testing.T) {
	dbMock, mock, store := setupMockDB(t)
	defer dbMock.Close()

	ctx := context.Background()

	mock.ExpectQuery(`(?i)SELECT\s+id,\s+password_hash\s+FROM\s+users\s+WHERE\s+username\s*=\s*\$1`).
		WithArgs("testuser").
		WillReturnError(errors.New("error"))

	_, _, err := store.UpdateSessionUser(ctx, "testuser", "password")
	assert.Error(t, err)
}

func TestUpdateSessionUserTxError(t *testing.T) {
	dbMock, mock, store := setupMockDB(t)
	defer dbMock.Close()

	ctx := context.Background()

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte("password"), bcrypt.DefaultCost)
	assert.NoError(t, err)

	mock.ExpectQuery(`(?i)SELECT\s+id,\s+password_hash\s+FROM\s+users\s+WHERE\s+username\s*=\s*\$1`).
		WithArgs("testuser").
		WillReturnRows(sqlmock.NewRows([]string{"id", "password_hash"}).
			AddRow("user-uid", string(hashedPassword)))

	mock.ExpectBegin()
	mock.ExpectExec(`(?i)INSERT\s+INTO\s+user_sessions\s*\(user_id, session_token, expires_at\)`).
		WithArgs("user-uid", sqlmock.AnyArg()).
		WillReturnError(errors.New("error"))
	mock.ExpectRollback()

	_, _, err = store.UpdateSessionUser(ctx, "testuser", "password")
	assert.Error(t, err)
}

func TestCheckSessionUser(t *testing.T) {
	dbMock, mock, store := setupMockDB(t)
	defer dbMock.Close()

	ctx := context.Background()
	mock.ExpectQuery(`(?i)SELECT\s+EXISTS\s*\(SELECT\s+1\s+FROM\s+user_sessions\s+WHERE\s+user_id\s*=\s*\$1\s+AND\s+session_token\s*=\s*\$2\)`).
		WithArgs("3a0a4950-16e3-4720-814b-17e6b4fd0bc2", "valid-token").
		WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))

	isValid := store.CheckSessionUser(ctx, "3a0a4950-16e3-4720-814b-17e6b4fd0bc2", "valid-token")
	assert.True(t, isValid)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestCheckSessionUserError(t *testing.T) {
	dbMock, mock, store := setupMockDB(t)
	defer dbMock.Close()

	ctx := context.Background()
	mock.ExpectQuery(`(?i)SELECT\s+EXISTS\s*\(SELECT\s+1\s+FROM\s+user_sessions\s+WHERE\s+user_id\s*=\s*\$1\s+AND\s+session_token\s*=\s*\$2\)`).
		WithArgs("user-uid", "valid-token").
		WillReturnError(errors.New("error"))

	isValid := store.CheckSessionUser(ctx, "user-uid", "valid-token")
	assert.False(t, isValid)
}

func TestGetDataNameList(t *testing.T) {
	dbMock, mock, store := setupMockDB(t)
	defer dbMock.Close()

	ctx := context.Background()
	mock.ExpectQuery(`(?i)SELECT\s+id,\s+data_type,\s+data_name\s+FROM\s+user_data\s+WHERE\s+user_id\s*=\s*\$1`).
		WithArgs("3a0a4950-16e3-4720-814b-17e6b4fd0bc1").
		WillReturnRows(sqlmock.NewRows([]string{"3a0a4950-16e3-4720-814b-17e6b4fd0bc2", "data_type", "data_name"}).
			AddRow("3a0a4950-16e3-4720-814b-17e6b4fd0bc3", "type1", "name1").
			AddRow("3a0a4950-16e3-4720-814b-17e6b4fd0bc4", "type2", "name2"))

	resp, err := store.GetDataNameList(ctx, "3a0a4950-16e3-4720-814b-17e6b4fd0bc1")
	assert.NoError(t, err)
	assert.Len(t, resp.Items, 2)
	assert.Equal(t, "3a0a4950-16e3-4720-814b-17e6b4fd0bc3", resp.Items[0].Id)
	assert.Equal(t, "type1", resp.Items[0].Type)
	assert.Equal(t, "name1", resp.Items[0].Name)
	assert.Equal(t, "3a0a4950-16e3-4720-814b-17e6b4fd0bc4", resp.Items[1].Id)
	assert.Equal(t, "type2", resp.Items[1].Type)
	assert.Equal(t, "name2", resp.Items[1].Name)
	assert.NoError(t, mock.ExpectationsWereMet())
}
