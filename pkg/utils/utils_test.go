package utils

import (
	"testing"

	"golang.org/x/crypto/bcrypt"
)

func TestHashPassword(t *testing.T) {
	t.Run("hash password", func(t *testing.T) {
		hash, err := HashPassword("test")
		if err != nil {
			t.Fatal(err)
		}
		err = bcrypt.CompareHashAndPassword([]byte(hash), []byte("test"))
	})
}
