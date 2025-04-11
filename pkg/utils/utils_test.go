package utils

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/stretchr/testify/assert"

	"golang.org/x/crypto/bcrypt"
)

func TestHashPassword(t *testing.T) {
	t.Run("hash password", func(t *testing.T) {
		hash, err := HashPassword("test")
		assert.NoError(t, err)
		err = bcrypt.CompareHashAndPassword([]byte(hash), []byte("test"))
		assert.NoError(t, err)
	})
}

func TestGenerateToken(t *testing.T) {
	t.Run("generate token", func(t *testing.T) {
		token, err := GenerateToken()
		assert.NoError(t, err)
		assert.NotEmpty(t, token)
	})
}

func TestHashPassword_Error(t *testing.T) {
	// сохраняем оригинал, чтобы потом вернуть
	original := generateFromPassword
	defer func() { generateFromPassword = original }()

	// подмена зависимости
	generateFromPassword = func(_ []byte, _ int) ([]byte, error) {
		return nil, errors.New("bcrypt fail")
	}

	_, err := HashPassword("test123")
	require.EqualError(t, err, "bcrypt fail")
}

func TestGenerateToken_Error(t *testing.T) {
	// сохраняем оригинал
	orig := randomReader
	defer func() { randomReader = orig }()

	// подменяем на фейл
	randomReader = func(_ []byte) (int, error) {
		return 0, errors.New("random fail")
	}

	_, err := GenerateToken()
	require.EqualError(t, err, "random fail")
}
