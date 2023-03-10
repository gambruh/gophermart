package hasher

import (
	"crypto/hmac"
	"crypto/sha256"
	"errors"
	"fmt"

	"github.com/gambruh/gophermart/internal/auth"
)

var ErrWrongPassword = errors.New("password is wrong")

func CalcHash(key string, str string) string {
	// Используя HMAC алгоритм, создаём подпись на основе предыдущей строки и ключа
	h := hmac.New(sha256.New, []byte(key))
	h.Write([]byte(str))
	sign := h.Sum(nil)

	return fmt.Sprintf("%x", sign)
}

func CheckHash(creds auth.LoginData, key string, hash string) error {
	hashtocheck := CalcHash(key, creds.Password)
	if hashtocheck != hash {
		return ErrWrongPassword
	}
	return nil
}
