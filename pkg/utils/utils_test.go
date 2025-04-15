package utils

import (
	"encoding/base64"
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

const validKey = "1234567890123456" // AES-128 требует 16 байт ключа

func TestEncryptDecrypt_Success(t *testing.T) {
	plain := "Hello"
	encrypted, err := Encrypt(plain, validKey)
	assert.NoError(t, err)
	assert.NotEmpty(t, encrypted)

	decrypted, err := Decrypt(encrypted, validKey)
	assert.NoError(t, err)
	assert.Equal(t, plain, decrypted)
}

func TestEncrypt_InvalidKeyLength(t *testing.T) {
	shortKey := "short"
	_, err := Encrypt("text", shortKey)
	assert.Error(t, err)
}

func TestDecrypt_InvalidBase64(t *testing.T) {
	_, err := Decrypt("!!!not-base64!!!", validKey)
	assert.Error(t, err)
}

func TestDecrypt_InvalidKeyLength(t *testing.T) {
	plain := "Test data"
	encrypted, err := Encrypt(plain, validKey)
	assert.NoError(t, err)

	_, err = Decrypt(encrypted, "badkey")
	assert.Error(t, err)
}

func TestDecrypt_InvalidCipherData(t *testing.T) {
	data := base64.StdEncoding.EncodeToString([]byte("short"))
	_, err := Decrypt(data, validKey)
	assert.Error(t, err)
}

func TestDecrypt_TamperedCipherText(t *testing.T) {
	plain := "Hello!"
	encrypted, err := Encrypt(plain, validKey)
	assert.NoError(t, err)

	// Порча зашифрованного текста
	tampered := encrypted + "123"
	_, err = Decrypt(tampered, validKey)
	assert.Error(t, err)
}
