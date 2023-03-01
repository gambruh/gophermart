package helpers

import (
	"time"

	"github.com/dgrijalva/jwt-go"
	"github.com/gambruh/gophermart/internal/config"
)

type LoginData struct {
	Login    string `json:"login"`
	Password string `json:"password"`
}

func GenerateToken(login string) (string, error) {
	// Create a new token object, specifying the signing method and the claims
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"login": login,
		"exp":   time.Now().Add(time.Hour * 24).Unix(),
	})

	// Sign the token with the secret key
	tokenString, err := token.SignedString([]byte(config.Cfg.Key))
	if err != nil {
		return "", err
	}

	return tokenString, nil
}
