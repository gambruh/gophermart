package auth

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/dgrijalva/jwt-go"
	"github.com/gambruh/gophermart/cmd/argon2id"
	"github.com/gambruh/gophermart/internal/config"
)

type LoginData struct {
	Login    string `json:"login"`
	Password string `json:"password"`
}

type AuthStorage interface {
	Register(login string, password string) error
	VerifyCredentials(login string, password string) error
	GetPass(username string) (string, error)
}

type AuthMemStorage struct {
	Data map[string]string
}

type AuthDB struct {
	db *sql.DB
}

// типы ошибок
var (
	ErrUserNotFound      = errors.New("user not found in database")
	ErrTableDoesntExist  = errors.New("table doesn't exist")
	ErrUsernameIsTaken   = errors.New("username is taken")
	ErrWrongCredentials  = errors.New("wrong login credentials")
	ErrWrongPassword     = errors.New("wrong password")
	ErrNoOperations      = errors.New("no records found")
	ErrInsufficientFunds = errors.New("not enough accrual to withdraw")
	ErrWrongOrder        = errors.New("wrong order")
)

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
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		token, err := jwt.ParseWithClaims(cookie.Value, &MyCustomClaims{}, func(token *jwt.Token) (interface{}, error) {
			return []byte(config.Cfg.Key), nil
		})
		if err != nil {
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

func NewAuthDB(postgresStr string) *AuthDB {
	db, _ := sql.Open("postgres", postgresStr)
	return &AuthDB{
		db: db,
	}
}

func GetAuthDB() (authstorage AuthStorage) {
	if config.Cfg.Storage {
		authstorage = NewMemStorage()
	} else {
		db := NewAuthDB(config.Cfg.Database)
		db.InitAuthDB()
		authstorage = db
	}

	return authstorage
}

func (s *AuthDB) CheckTableExists(tablename string) error {
	var check bool

	err := s.db.QueryRow(checkTableExistsQuery, tablename).Scan(&check)
	if err != nil {
		log.Printf("error checking if table exists: %v", err)
		return err
	}
	if !check {
		return ErrTableDoesntExist
	}
	return nil
}

func (s *AuthDB) InitAuthDB() error {
	err := s.CheckNDropTables()
	if err != nil {
		return err
	}
	err = s.CreateUsersTable()
	if err != nil {
		return err
	}
	err = s.CreatePasswordsTable()
	if err != nil {
		return err
	}
	return nil
}
func (s *AuthDB) CreateUsersTable() error {
	err := s.CheckTableExists("users")
	if err == ErrTableDoesntExist {
		_, err := s.db.Exec(createUsersTableQuery)
		if err != nil {
			return err
		}
	}
	return nil
}
func (s *AuthDB) CreatePasswordsTable() error {
	err := s.CheckTableExists("passwords")
	if err == ErrTableDoesntExist {
		_, err = s.db.Exec(createPasswordsTableQuery)
		if err != nil {
			log.Println("error when creating passwords table:", err)
			return err
		}
	}
	return nil
}

func (s *AuthDB) CheckNDropTables() error {
	var check bool

	err := s.db.QueryRow(checkTableExistsQuery, "users").Scan(&check)
	if err != nil {
		fmt.Printf("error checking if table exists: %v", err)
		return err
	}
	if check {
		_, err = s.db.Exec(dropPasswordsTableQuery)
		if err != nil {
			return err
		}
		_, err = s.db.Exec(dropUsersTableQuery)
		if err != nil {
			return err
		}
	}
	if err != nil {
		fmt.Printf("error dropping a table: %v", err)
		return err
	}

	return nil
}

func (s *AuthDB) Register(login string, password string) error {
	var username string
	hashedpassword, err := argon2id.CreateHash(password, argon2id.DefaultParams)
	if err != nil {
		log.Println("error when trying to hash password:", err)
		return err
	}
	e := s.db.QueryRow(CheckUsernameQuery, login).Scan(&username)

	switch e {
	case sql.ErrNoRows:
		_, err := s.db.Exec(AddNewUserQuery, login, hashedpassword)
		return err
	case nil:
		err := ErrUsernameIsTaken
		return err
	default:
		fmt.Printf("Something wrong when adding a user in database: %v\n", e)
		return e
	}
}

func (s *AuthDB) VerifyCredentials(login string, password string) error {
	var (
		id   int
		pass string
	)

	err := s.db.QueryRow(CheckUsernameQuery, login).Scan(&id)
	switch err {
	case sql.ErrNoRows:
		return nil
	case nil:
	default:
		log.Println("Unexpected case in checking user's credentials in database:", err)
		return err
	}

	err = s.db.QueryRow(CheckPasswordQuery, id).Scan(&pass)
	if err != nil {
		log.Println("Unexpected case in checking user's password in database:", err)
		return err
	}

	check, err := argon2id.ComparePasswordAndHash(password, pass)
	if err != nil {
		log.Println("error when trying to compare password and hash:", err)
		return err
	}
	if !check {
		return ErrWrongPassword
	}

	return nil
}

func (s *AuthMemStorage) Register(login string, password string) error {
	_, contains := s.Data[login]
	if contains {
		return ErrUsernameIsTaken
	}
	s.Data[login] = password
	return nil
}

func (s AuthMemStorage) VerifyCredentials(login string, password string) error {
	check, contains := s.Data[login]
	if contains && check == password {
		return nil
	}
	return ErrWrongPassword
}

func NewMemStorage() *AuthMemStorage {
	return &AuthMemStorage{
		Data: make(map[string]string),
	}
}

func (s AuthMemStorage) GetPass(username string) (string, error) {
	password, contains := s.Data[username]
	if !contains {
		return "", ErrUserNotFound
	}

	return password, nil
}

func (s *AuthDB) GetPass(username string) (string, error) {
	var password string
	err := s.db.QueryRow(getPassQuery, username).Scan(&password)
	if err != nil {
		return "", err
	}

	return password, nil
}
