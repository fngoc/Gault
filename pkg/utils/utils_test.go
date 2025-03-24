package utils

import (
	"testing"

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
