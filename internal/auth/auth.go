package auth

import (
	"context"
	"net/http"
	"time"

	"github.com/dgrijalva/jwt-go"
	"github.com/gambruh/gophermart/internal/config"
)

// auth queries
var CheckUsernameQuery = `
	SELECT id
	FROM users 
	WHERE username = $1;
`

var CheckPasswordQuery = `
	SELECT password
	FROM passwords 
	WHERE id = $1;
`

var AddNewUserQuery = `
	WITH new_user AS (
		INSERT INTO users (username)
		VALUES ($1)
		RETURNING id
	)
	INSERT INTO passwords (id, password)
	VALUES ((SELECT id FROM new_user), $2);
`

type LoginData struct {
	Login    string `json:"login"`
	Password string `json:"password"`
}

func GenerateToken(login string) (string, error) {
	// Create a new token object, specifying the signing method and the claims
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"userID": login,
		"exp":    time.Now().Add(time.Hour * 8).Unix(),
	})

	// Sign the token with the secret key
	tokenString, err := token.SignedString([]byte(config.Cfg.Key))
	if err != nil {
		return "", err
	}

	return tokenString, nil
}

func AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		type MyCustomClaims struct {
			UserID string `json:"userID"`
			jwt.StandardClaims
		}

		cookie, err := r.Cookie("gophermart-auth")
		if err != nil {
			//fmt.Println("can't get cookie!")
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		token, err := jwt.ParseWithClaims(cookie.Value, &MyCustomClaims{}, func(token *jwt.Token) (interface{}, error) {
			return []byte(config.Cfg.Key), nil
		})
		if err != nil {
			//fmt.Println("error when parsing token!")
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		claims, ok := token.Claims.(*MyCustomClaims)

		if !ok || !token.Valid {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		ctx := context.WithValue(r.Context(), config.UserID("userID"), claims.UserID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
