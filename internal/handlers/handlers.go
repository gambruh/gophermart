package handlers

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"

	"github.com/gambruh/gophermart/internal/database"
	"github.com/gambruh/gophermart/internal/helpers"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

type WebService struct {
	Database *database.SQLdb
	Mu       *sync.Mutex
}

func (h *WebService) Service() http.Handler {

	r := chi.NewRouter()
	r.Use(middleware.Compress(5, "text/plain", "text/html", "application/json"))
	r.Route("/", func(r chi.Router) {
		r.Post("/api/user/register", h.Register)
		r.Post("/api/user/login", h.Login)
	})

	return r
}

func NewService(database *database.SQLdb) *WebService {
	return &WebService{
		Database: database,
		Mu:       &sync.Mutex{},
	}
}

func (h *WebService) Register(w http.ResponseWriter, r *http.Request) {
	// Проверка запроса на валидность - структура json. Вернуть 400, если запрос неправильный.
	var data helpers.LoginData
	err := json.NewDecoder(r.Body).Decode(&data)
	if err != nil {
		log.Printf("Wrong format of incoming json request: %v\n", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	if data.Login == "" {
		log.Println("Empty login field")
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	if data.Password == "" {
		log.Println("Empty password field")
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	//пока обращение к базе данных
	err = h.Database.Register(data.Login, data.Password)
	switch err {
	case database.ErrUsernameIsTaken:
		log.Println("Username is taken")
		w.WriteHeader(http.StatusConflict)
	case nil:
		// Generate a token or session ID
		token, err := helpers.GenerateToken(data.Login)
		if err != nil {
			fmt.Println("error when generating token", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		// Set the token as a cookie in the response
		http.SetCookie(w, &http.Cookie{
			Name:  "token",
			Value: token,
		})

		// Return a success response
		w.WriteHeader(http.StatusOK)
	default:
		log.Println("Unexpected case in new user registration:", err)
		w.WriteHeader(http.StatusInternalServerError)
	}
}

func (h *WebService) Login(w http.ResponseWriter, r *http.Request) {
	// Parse the request body into a LoginData struct
	var data helpers.LoginData
	err := json.NewDecoder(r.Body).Decode(&data)
	if err != nil {
		log.Println("Wrong login credentials format:", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	// Verify the user's credentials
	check, err := h.Database.VerifyCredentials(data.Login, data.Password)
	if err != nil {
		fmt.Println("error when verifying login credentials:", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	if !check {
		fmt.Println("Invalid login credentials:", data.Login)
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	// Generate a token or session ID
	token, err := helpers.GenerateToken(data.Login)
	if err != nil {
		fmt.Println("error when generating token", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	// Set the token as a cookie in the response
	http.SetCookie(w, &http.Cookie{
		Name:  "token",
		Value: token,
	})

	// Return a success response
	w.WriteHeader(http.StatusOK)
}
