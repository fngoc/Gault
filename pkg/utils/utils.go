package utils

import (
	"crypto/rand"
	"encoding/base64"

	"golang.org/x/crypto/bcrypt"
)

// объявляем переменную, которую можно подменить в тестах
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
