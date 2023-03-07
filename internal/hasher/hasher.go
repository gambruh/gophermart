package hasher

import (
	"crypto/hmac"
	"crypto/sha256"
	"fmt"

	"github.com/gambruh/gophermart/internal/auth"
)

func CalcHash(key string, str string) string {
	// Используя HMAC алгоритм, создаём подпись на основе предыдущей строки и ключа
	h := hmac.New(sha256.New, []byte(key))
	h.Write([]byte(str))
	sign := h.Sum(nil)

	return fmt.Sprintf("%x", sign)
}

func CheckHash(creds auth.LoginData, key string) bool {
	//TODO
	//вычислить хэш пароля из входящей структуры
	//обратиться к базе данных, запросить строку хэша из базы
	//если строки совпадают - вернуть ок, продолжить аутентификацию
	//если не сопадают - вернуть ошибку
	return false
}
