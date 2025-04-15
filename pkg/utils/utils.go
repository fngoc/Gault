package utils

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"

	"golang.org/x/crypto/bcrypt"
)

// Объявляем переменную, которую можно подменить в тестах
var generateFromPassword = bcrypt.GenerateFromPassword

// HashPassword хеширование пароля
func HashPassword(password string) (string, error) {
	hashedBytes, err := generateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(hashedBytes), nil
}

// переменная для замены в тестах
var randomReader = rand.Read

// GenerateToken создает случайный токен длиной 32 байта (256 бит)
func GenerateToken() (string, error) {
	bytes := make([]byte, 32)
	_, err := randomReader(bytes)
	if err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(bytes), nil
}

// Encrypt шифрует строку с помощью AES-GCM и возвращает base64-строку
func Encrypt(plainText, encryptionKey string) (string, error) {
	block, err := aes.NewCipher([]byte(encryptionKey))
	if err != nil {
		return "", err
	}

	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	nonce := make([]byte, aesGCM.NonceSize())
	if _, err = io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}

	cipherText := aesGCM.Seal(nonce, nonce, []byte(plainText), nil)
	return base64.StdEncoding.EncodeToString(cipherText), nil
}

// Decrypt расшифровывает base64-строку обратно в оригинальную строку
func Decrypt(enc, encryptionKey string) (string, error) {
	cipherData, err := base64.StdEncoding.DecodeString(enc)
	if err != nil {
		return "", err
	}

	block, err := aes.NewCipher([]byte(encryptionKey))
	if err != nil {
		return "", err
	}

	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	nonceSize := aesGCM.NonceSize()
	if len(cipherData) < nonceSize {
		return "", fmt.Errorf("invalid cipher data")
	}

	nonce, cipherText := cipherData[:nonceSize], cipherData[nonceSize:]
	plainText, err := aesGCM.Open(nil, nonce, cipherText, nil)
	if err != nil {
		return "", err
	}

	return string(plainText), nil
}
