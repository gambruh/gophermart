package handlers

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"sync"

	"github.com/gambruh/gophermart/internal/auth"
	"github.com/gambruh/gophermart/internal/balance"
	"github.com/gambruh/gophermart/internal/config"
	"github.com/gambruh/gophermart/internal/database"
	"github.com/gambruh/gophermart/internal/orders"
	"github.com/gambruh/gophermart/internal/storage"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

type WebService struct {
	Storage storage.Storage
	Mu      *sync.Mutex
}

var ErrWrongCredentials = errors.New("wrong login/password")
var ErrUsernameIsTaken = errors.New("username is taken")

func (h *WebService) Service() http.Handler {

	r := chi.NewRouter()
	r.Use(middleware.Compress(5, "text/plain", "text/html", "application/json"))

	r.Post("/api/user/register", h.Register)
	r.Post("/api/user/login", h.Login)

	r.Group(func(r chi.Router) {
		r.Use(auth.AuthMiddleware)
		r.Post("/api/user/orders", h.PostOrder)
		r.Get("/api/user/orders", h.GetOrders)
		r.Get("/api/user/balance", h.GetBalance)
		r.Post("/api/user/balance/withdraw", h.Withdraw)
		r.Get("/api/user/withdrawals", h.GetWithdrawals)
	})

	return r
}

func NewService(storage storage.Storage) *WebService {
	return &WebService{
		Storage: storage,
		Mu:      &sync.Mutex{},
	}
}

func (h *WebService) Register(w http.ResponseWriter, r *http.Request) {
	// Проверка запроса на валидность - структура json. Вернуть 400, если запрос неправильный.
	var data auth.LoginData
	err := json.NewDecoder(r.Body).Decode(&data)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	if data.Login == "" {
		w.Write([]byte("Empty login field"))
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	if data.Password == "" {

		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("Empty password field"))
		return
	}

	//пока обращение к базе данных
	err = h.Storage.Register(data.Login, data.Password)
	switch err {
	case database.ErrUsernameIsTaken, storage.ErrUsernameIsTaken:
		fmt.Println("Username is taken")
		w.WriteHeader(http.StatusConflict)
		return
	case nil:
		// Generate token
		token, err := auth.GenerateToken(data.Login)
		if err != nil {
			fmt.Println("error when generating token", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		// Set the token in "Cookies"
		http.SetCookie(w, &http.Cookie{
			Name:  "gophermart-auth",
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
	var data auth.LoginData
	err := json.NewDecoder(r.Body).Decode(&data)
	if err != nil {
		log.Println("Wrong login credentials format:", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	// Verify the user's credentials
	err = h.Storage.VerifyCredentials(data.Login, data.Password)
	switch {
	case err == nil:
		//login and password are verified
	case err.Error() == ErrWrongCredentials.Error():
		fmt.Println("Invalid login credentials:", data.Login)
		w.WriteHeader(http.StatusUnauthorized)
		return
	default:
		fmt.Println("error when verifying login credentials:", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	// Generate a token
	token, err := auth.GenerateToken(data.Login)
	if err != nil {
		fmt.Println("error when generating token", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	// Set the token as a cookie in the response
	http.SetCookie(w, &http.Cookie{
		Name:  "gophermart-auth",
		Value: token,
	})

	// Return a success response
	w.WriteHeader(http.StatusOK)
}

func (h *WebService) PostOrder(w http.ResponseWriter, r *http.Request) {
	contentType := r.Header.Get("Content-type")
	if contentType != "text/plain" {
		fmt.Printf("content-type is not text/plain, but %s\n", contentType)
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	username := r.Context().Value(config.UserID("userID"))

	body, err := io.ReadAll(r.Body)
	if err != nil {
		log.Println("error when trying to read request body in PostOrder handler:", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	defer r.Body.Close()
	ordernumber := string(body)
	//check if the order is valid by Luhn's algo
	if !orders.LuhnCheck(ordernumber) {
		w.WriteHeader(http.StatusUnprocessableEntity)
		return
	}
	w.Header().Add("Content-type", "application/json")
	//attempt to write a new order into storage
	err = h.Storage.SetOrder(ordernumber, username.(string))
	switch err {
	case nil:
		w.WriteHeader(http.StatusAccepted)
	case orders.ErrOrderLoadedThisUser:
		w.WriteHeader(http.StatusOK)
	case orders.ErrOrderLoadedAnotherUser:
		w.WriteHeader(http.StatusConflict)
	default:
		log.Println("Unexpected case in PostOrder Handler:", err)
		w.WriteHeader(http.StatusInternalServerError)
	}
}

func (h *WebService) GetOrders(w http.ResponseWriter, r *http.Request) {
	ords, err := h.Storage.GetOrders(r.Context())
	w.Header().Add("Content-type", "application/json")
	switch err {
	case nil:
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(ords)
	case orders.ErrNoOrders:
		w.WriteHeader(http.StatusNoContent)
	default:
		log.Println("error in GetOrders handler:", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
}

func (h *WebService) GetBalance(w http.ResponseWriter, r *http.Request) {
	bal, err := h.Storage.GetBalance(r.Context())
	switch err {
	case nil:
		w.Header().Add("Content-type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(bal)
		fmt.Println("balance is:", bal)
	default:
		log.Println("error in GetBalance handler:", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
}

func (h *WebService) GetWithdrawals(w http.ResponseWriter, r *http.Request) {
	withdrawals, err := h.Storage.GetWithdrawals(r.Context())
	switch err {
	case nil:
		w.Header().Add("Content-type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(withdrawals)
	case balance.ErrNoOperations:
		w.WriteHeader(http.StatusNoContent)
	default:
		log.Println("error in GetWithdrawals handler:", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
}

func (h *WebService) Withdraw(w http.ResponseWriter, r *http.Request) {
	var withdrawReq balance.WithdrawQ

	err := json.NewDecoder(r.Body).Decode(&withdrawReq)
	if err != nil {
		log.Println("error in Withdraw handler:", err)
		w.WriteHeader(http.StatusInternalServerError)
	}

	err = h.Storage.Withdraw(r.Context(), withdrawReq)
	switch err {
	case nil:
		w.WriteHeader(http.StatusOK)
	case balance.ErrWrongOrder:
		w.WriteHeader(http.StatusUnprocessableEntity)
	case balance.ErrInsufficientFunds:
		w.WriteHeader(http.StatusPaymentRequired)
	default:
		log.Println("error in Withdraw handler when adding withdraw operation in storage:", err)
		w.WriteHeader(http.StatusInternalServerError)
	}
}
