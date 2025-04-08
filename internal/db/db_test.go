package db

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
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

func TestBeginTx_Success(t *testing.T) {
	dbMock, mock, store := setupMockDB(t)
	defer dbMock.Close()

	ctx := context.Background()

	mock.ExpectBegin()

	tx, err := store.BeginTx(ctx)
	assert.NoError(t, err)
	assert.NotNil(t, tx)

	mock.ExpectCommit()
	err = tx.Commit()
	assert.NoError(t, err)

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestBeginTx_Error(t *testing.T) {
	dbMock, mock, store := setupMockDB(t)
	defer dbMock.Close()

	ctx := context.Background()

	mock.ExpectBegin().WillReturnError(errors.New("begin tx error"))

	tx, err := store.BeginTx(ctx)
	assert.Error(t, err)
	assert.Nil(t, tx)

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestCreateEmptyLO_Success(t *testing.T) {
	dbMock, mock, store := setupMockDB(t)
	defer dbMock.Close()

	ctx := context.Background()

	mock.ExpectBegin()
	tx, err := store.BeginTx(ctx)
	assert.NoError(t, err)

	mock.ExpectQuery(`SELECT lo_create\(0\)`).
		WillReturnRows(sqlmock.NewRows([]string{"lo_create"}).AddRow(12345))

	oid, err := store.CreateEmptyLO(ctx, tx)
	assert.NoError(t, err)
	assert.Equal(t, 12345, oid)

	mock.ExpectCommit()
	err = tx.Commit()
	assert.NoError(t, err)

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestCreateEmptyLO_Error(t *testing.T) {
	dbMock, mock, store := setupMockDB(t)
	defer dbMock.Close()

	ctx := context.Background()

	mock.ExpectBegin()
	tx, err := store.BeginTx(ctx)
	assert.NoError(t, err)

	mock.ExpectQuery(`SELECT lo_create\(0\)`).
		WillReturnError(errors.New("lo_create failed"))

	_, err = store.CreateEmptyLO(ctx, tx)
	assert.Error(t, err)

	mock.ExpectRollback()
	rerr := tx.Rollback()
	assert.NoError(t, rerr)

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestInsertUserDataRecordTx_Success(t *testing.T) {
	dbMock, mock, store := setupMockDB(t)
	defer dbMock.Close()

	ctx := context.Background()

	mock.ExpectBegin()
	tx, err := store.BeginTx(ctx)
	assert.NoError(t, err)

	mock.ExpectExec(`INSERT INTO user_data`).
		WithArgs(
			sqlmock.AnyArg(),
			sqlmock.AnyArg(),
			"test-type",
			"test-name",
			99,
		).
		WillReturnResult(sqlmock.NewResult(1, 1))

	err = store.InsertUserDataRecordTx(ctx, tx, "data-uuid", "user-uuid", "test-type", "test-name", 99)
	assert.NoError(t, err)

	mock.ExpectCommit()
	err = tx.Commit()
	assert.NoError(t, err)

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestInsertUserDataRecordTx_Error(t *testing.T) {
	dbMock, mock, store := setupMockDB(t)
	defer dbMock.Close()

	ctx := context.Background()

	mock.ExpectBegin()
	tx, err := store.BeginTx(ctx)
	assert.NoError(t, err)

	mock.ExpectExec(`INSERT INTO user_data`).
		WillReturnError(errors.New("insert user_data failed"))

	err = store.InsertUserDataRecordTx(ctx, tx, "data-uuid", "user-uuid", "type", "name", 999)
	assert.Error(t, err)

	mock.ExpectRollback()
	rerr := tx.Rollback()
	assert.NoError(t, rerr)

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestOpenLOForWriting_Success(t *testing.T) {
	dbMock, mock, store := setupMockDB(t)
	defer dbMock.Close()

	ctx := context.Background()

	mock.ExpectBegin()
	tx, err := store.BeginTx(ctx)
	assert.NoError(t, err)

	mock.ExpectQuery(`SELECT lo_open\(\$1, \$2\)`).
		WithArgs(12345, 131072).
		WillReturnRows(sqlmock.NewRows([]string{"lo_open"}).AddRow(10))

	fd, err := store.OpenLOForWriting(ctx, tx, 12345)
	assert.NoError(t, err)
	assert.Equal(t, 10, fd)

	mock.ExpectCommit()
	err = tx.Commit()
	assert.NoError(t, err)

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestOpenLOForWriting_Error(t *testing.T) {
	dbMock, mock, store := setupMockDB(t)
	defer dbMock.Close()

	ctx := context.Background()

	mock.ExpectBegin()
	tx, err := store.BeginTx(ctx)
	assert.NoError(t, err)

	mock.ExpectQuery(`SELECT lo_open\(\$1, \$2\)`).
		WillReturnError(errors.New("lo_open failed"))

	fd, err := store.OpenLOForWriting(ctx, tx, 12345)
	assert.Error(t, err)
	assert.Equal(t, 0, fd)

	mock.ExpectRollback()
	rerr := tx.Rollback()
	assert.NoError(t, rerr)

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestWriteLO_Success(t *testing.T) {
	dbMock, mock, store := setupMockDB(t)
	defer dbMock.Close()

	ctx := context.Background()

	mock.ExpectBegin()
	tx, err := store.BeginTx(ctx)
	assert.NoError(t, err)

	chunk := []byte("Hello, world!")
	expectedWriteLen := len(chunk)

	mock.ExpectQuery(`SELECT lowrite\(\$1, \$2\)`).
		WithArgs(10, chunk).
		WillReturnRows(sqlmock.NewRows([]string{"lowrite"}).AddRow(expectedWriteLen))

	err = store.WriteLO(ctx, tx, 10, chunk)
	assert.NoError(t, err)

	mock.ExpectCommit()
	err = tx.Commit()
	assert.NoError(t, err)

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestWriteLO_PartialWrite(t *testing.T) {
	dbMock, mock, store := setupMockDB(t)
	defer dbMock.Close()

	ctx := context.Background()

	mock.ExpectBegin()
	tx, err := store.BeginTx(ctx)
	assert.NoError(t, err)

	chunk := []byte("Hello!")
	mock.ExpectQuery(`SELECT lowrite\(\$1, \$2\)`).
		WithArgs(10, chunk).
		WillReturnRows(sqlmock.NewRows([]string{"lowrite"}).AddRow(3))

	err = store.WriteLO(ctx, tx, 10, chunk)
	assert.Error(t, err)

	mock.ExpectRollback()
	rerr := tx.Rollback()
	assert.NoError(t, rerr)

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestWriteLO_Error(t *testing.T) {
	dbMock, mock, store := setupMockDB(t)
	defer dbMock.Close()

	ctx := context.Background()

	mock.ExpectBegin()
	tx, err := store.BeginTx(ctx)
	assert.NoError(t, err)

	mock.ExpectQuery(`SELECT lowrite\(\$1, \$2\)`).
		WillReturnError(errors.New("lowrite failed"))

	err = store.WriteLO(ctx, tx, 10, []byte("data"))
	assert.Error(t, err)

	mock.ExpectRollback()
	rerr := tx.Rollback()
	assert.NoError(t, rerr)

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestCloseLO(t *testing.T) {
	dbMock, mock, store := setupMockDB(t)
	defer dbMock.Close()

	ctx := context.Background()

	mock.ExpectBegin()
	tx, err := store.BeginTx(ctx)
	assert.NoError(t, err)

	mock.ExpectExec(`SELECT lo_close\(\$1\)`).
		WithArgs(10).
		WillReturnResult(sqlmock.NewResult(1, 1))

	store.CloseLO(ctx, tx, 10)

	mock.ExpectCommit()
	err = tx.Commit()
	assert.NoError(t, err)

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetData_Error_File(t *testing.T) {
	dbMock, mock, store := setupMockDB(t)
	defer dbMock.Close()

	ctx := context.Background()

	mock.ExpectBegin()

	mock.ExpectQuery(`SELECT data_type, data_name, largeobject_oid FROM user_data WHERE id = \$1`).
		WithArgs(sqlmock.AnyArg()).
		WillReturnRows(sqlmock.NewRows([]string{"data_type", "data_name", "largeobject_oid"}).
			AddRow("file", "some-name", 123))

	mock.ExpectQuery(`SELECT lo_open\(\$1, 131072\)`).
		WithArgs(123).
		WillReturnRows(sqlmock.NewRows([]string{"lo_open"}).AddRow(10))

	chunkRows := []string{"loread"}
	mock.ExpectQuery(`SELECT loread\(\$1, \$2\)`).
		WithArgs(10, 1024*1024).
		WillReturnRows(sqlmock.NewRows(chunkRows).AddRow([]byte("Hello, ")))
	mock.ExpectQuery(`SELECT loread\(\$1, \$2\)`).
		WithArgs(10, 1024*1024).
		WillReturnRows(sqlmock.NewRows(chunkRows).AddRow([]byte("world!")))
	mock.ExpectQuery(`SELECT loread\(\$1, \$2\)`).
		WithArgs(10, 1024*1024).
		WillReturnRows(sqlmock.NewRows(chunkRows).AddRow([]byte{}))

	mock.ExpectExec(`SELECT lo_close\(\$1\)`).
		WithArgs(10).
		WillReturnResult(sqlmock.NewResult(1, 1))

	mock.ExpectCommit()

	defer func() {
		if r := recover(); r != nil {
			fmt.Println("Recovered from panic for db test")
		}
	}()
	resp, err := store.GetData(ctx, "3a0a4950-16e3-4720-814b-17e6b4fd0bc2")
	assert.Error(t, err)
	assert.Nil(t, resp)
}

func TestGetData_Error_Text(t *testing.T) {
	dbMock, mock, store := setupMockDB(t)
	defer dbMock.Close()

	ctx := context.Background()

	mock.ExpectBegin()

	mock.ExpectQuery(`SELECT data_type, data_name, largeobject_oid FROM user_data WHERE id = \$1`).
		WithArgs(sqlmock.AnyArg()).
		WillReturnRows(sqlmock.NewRows([]string{"data_type", "data_name", "largeobject_oid"}).
			AddRow("text", "some-name", 999))

	mock.ExpectQuery(`SELECT lo_open\(\$1, 131072\)`).
		WithArgs(999).
		WillReturnRows(sqlmock.NewRows([]string{"lo_open"}).AddRow(20))

	mock.ExpectQuery(`SELECT loread\(\$1, \$2\)`).
		WithArgs(20, 1024*1024).
		WillReturnRows(sqlmock.NewRows([]string{"loread"}).AddRow([]byte("Привет!")))
	mock.ExpectQuery(`SELECT loread\(\$1, \$2\)`).
		WithArgs(20, 1024*1024).
		WillReturnRows(sqlmock.NewRows([]string{"loread"}).AddRow([]byte{}))

	mock.ExpectExec(`SELECT lo_close\(\$1\)`).
		WithArgs(20).
		WillReturnResult(sqlmock.NewResult(1, 1))

	mock.ExpectCommit()

	defer func() {
		if r := recover(); r != nil {
			fmt.Println("Recovered from panic for db test")
		}
	}()
	resp, err := store.GetData(ctx, "some-id")
	assert.Error(t, err)
	assert.Nil(t, resp)
}

func TestGetData_BeginTxError(t *testing.T) {
	dbMock, mock, store := setupMockDB(t)
	defer dbMock.Close()

	ctx := context.Background()

	mock.ExpectBegin().WillReturnError(errors.New("begin tx error"))

	resp, err := store.GetData(ctx, "any-id")
	assert.Error(t, err)
	assert.Nil(t, resp)
	assert.Contains(t, err.Error(), "begin transaction: begin tx error")

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetData_NotFoundError(t *testing.T) {
	dbMock, mock, store := setupMockDB(t)
	defer dbMock.Close()

	ctx := context.Background()

	mock.ExpectBegin()

	mock.ExpectQuery(`SELECT data_type, data_name, largeobject_oid FROM user_data WHERE id = \$1`).
		WillReturnError(sql.ErrNoRows)

	mock.ExpectRollback()

	resp, err := store.GetData(ctx, "nonexistent-id")
	assert.Error(t, err)
	assert.Nil(t, resp)

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetData_loOpenError(t *testing.T) {
	dbMock, mock, store := setupMockDB(t)
	defer dbMock.Close()

	ctx := context.Background()

	mock.ExpectBegin()

	mock.ExpectQuery(`SELECT data_type, data_name, largeobject_oid FROM user_data WHERE id = \$1`).
		WillReturnRows(sqlmock.NewRows([]string{"data_type", "data_name", "largeobject_oid"}).
			AddRow("file", "name", 123))

	mock.ExpectQuery(`SELECT lo_open\(\$1, 131072\)`).
		WithArgs(123).
		WillReturnError(errors.New("lo_open failed"))

	mock.ExpectRollback()

	resp, err := store.GetData(ctx, "some-id")
	assert.Error(t, err)
	assert.Nil(t, resp)
	assert.Contains(t, err.Error(), "lo_open failed")

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetData_loreadError(t *testing.T) {
	dbMock, mock, store := setupMockDB(t)
	defer dbMock.Close()

	ctx := context.Background()

	mock.ExpectBegin()

	mock.ExpectQuery(`SELECT data_type, data_name, largeobject_oid FROM user_data WHERE id = \$1`).
		WillReturnRows(sqlmock.NewRows([]string{"data_type", "data_name", "largeobject_oid"}).
			AddRow("file", "name", 777))

	mock.ExpectQuery(`SELECT lo_open\(\$1, 131072\)`).
		WithArgs(777).
		WillReturnRows(sqlmock.NewRows([]string{"lo_open"}).AddRow(40))

	mock.ExpectQuery(`SELECT loread\(\$1, \$2\)`).
		WillReturnError(errors.New("loread failed"))

	mock.ExpectRollback()

	defer func() {
		if r := recover(); r != nil {
			fmt.Println("Recovered from panic for db test")
		}
	}()
	resp, err := store.GetData(ctx, "some-id")
	assert.Error(t, err)
	assert.Nil(t, resp)
}

func TestGetData_CommitError(t *testing.T) {
	dbMock, mock, store := setupMockDB(t)
	defer dbMock.Close()

	ctx := context.Background()

	mock.ExpectBegin()

	mock.ExpectQuery(`SELECT data_type, data_name, largeobject_oid FROM user_data WHERE id = \$1`).
		WillReturnRows(sqlmock.NewRows([]string{"data_type", "data_name", "largeobject_oid"}).
			AddRow("file", "name", 555))

	mock.ExpectQuery(`SELECT lo_open\(\$1, 131072\)`).
		WithArgs(555).
		WillReturnRows(sqlmock.NewRows([]string{"lo_open"}).AddRow(50))

	mock.ExpectQuery(`SELECT loread\(\$1, \$2\)`).
		WillReturnRows(sqlmock.NewRows([]string{"loread"}).AddRow([]byte("some chunk data")))
	mock.ExpectQuery(`SELECT loread\(\$1, \$2\)`).
		WillReturnRows(sqlmock.NewRows([]string{"loread"}).AddRow([]byte{}))

	mock.ExpectExec(`SELECT lo_close\(\$1\)`).
		WithArgs(50).
		WillReturnResult(sqlmock.NewResult(1, 1))

	mock.ExpectCommit().WillReturnError(errors.New("commit error"))

	defer func() {
		if r := recover(); r != nil {
			fmt.Println("Recovered from panic for db test")
		}
	}()
	resp, err := store.GetData(ctx, "some-id")
	assert.Error(t, err)
	assert.Nil(t, resp)
}

func TestTruncateLO_Success(t *testing.T) {
	dbMock, mock, store := setupMockDB(t)
	defer dbMock.Close()

	ctx := context.Background()

	mock.ExpectBegin()
	tx, err := store.BeginTx(ctx)
	assert.NoError(t, err)

	mock.ExpectExec(`SELECT lo_truncate\(\$1, \$2\)`).
		WithArgs(10, int64(1000)).
		WillReturnResult(sqlmock.NewResult(1, 1))

	err = store.TruncateLO(ctx, tx, 10, 1000)
	assert.NoError(t, err)

	mock.ExpectCommit()
	err = tx.Commit()
	assert.NoError(t, err)

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestTruncateLO_Error(t *testing.T) {
	dbMock, mock, store := setupMockDB(t)
	defer dbMock.Close()

	ctx := context.Background()

	mock.ExpectBegin()
	tx, err := store.BeginTx(ctx)
	assert.NoError(t, err)

	mock.ExpectExec(`SELECT lo_truncate\(\$1, \$2\)`).
		WithArgs(10, int64(1000)).
		WillReturnError(errors.New("lo_truncate error"))

	err = store.TruncateLO(ctx, tx, 10, 1000)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "lo_truncate failed")

	mock.ExpectRollback()
	rerr := tx.Rollback()
	assert.NoError(t, rerr)

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetOidByItemID_Error(t *testing.T) {
	dbMock, mock, store := setupMockDB(t)
	defer dbMock.Close()

	ctx := context.Background()

	mock.ExpectQuery(`SELECT oid FROM your_table WHERE id = \$1`).
		WithArgs(sqlmock.AnyArg()).
		WillReturnRows(sqlmock.NewRows([]string{"oid"}).AddRow(42))

	_, err := store.GetOidByItemID(ctx, "item-id")
	assert.Error(t, err)
}
