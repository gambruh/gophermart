package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/gambruh/gophermart/internal/auth"
	"github.com/gambruh/gophermart/internal/config"
	"github.com/gambruh/gophermart/internal/database"
)

func TestWebService_Register(t *testing.T) {
	tests := []struct {
		name      string
		h         *WebService
		loginData auth.LoginData
		want      int
	}{
		{
			name: "test 1 write login data to storage",
			h: &WebService{
				Storage:     &database.MemStorage{Data: make(map[string]string)},
				AuthStorage: &auth.AuthMemStorage{Data: make(map[string]string)},
				Mu:          &sync.Mutex{},
			},
			loginData: auth.LoginData{
				Login:    "user123",
				Password: "secretpass",
			},
			want: http.StatusOK,
		},
		{
			name: "test 2 empty password",
			h: &WebService{
				Storage: &database.MemStorage{Data: make(map[string]string)},
				Mu:      &sync.Mutex{},
			},
			loginData: auth.LoginData{
				Login:    "user123",
				Password: "",
			},
			want: http.StatusBadRequest,
		},
		{
			name: "test 3 username already exists",
			h: &WebService{
				Storage:     &database.MemStorage{Data: map[string]string{"user123": "secretpass"}},
				AuthStorage: &auth.AuthMemStorage{Data: map[string]string{"user123": "secretpass"}},
				Mu:          &sync.Mutex{},
			},
			loginData: auth.LoginData{
				Login:    "user123",
				Password: "secretpass1",
			},
			want: http.StatusConflict,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			//подготовка боди - json структуры в формате loginData
			rbody, err := json.Marshal(tt.loginData)
			if err != nil {
				t.Fatal(err)
			}

			rr := httptest.NewRecorder()
			req, err := http.NewRequest(http.MethodPost, "/api/user/register", bytes.NewBuffer(rbody))
			if err != nil {
				t.Fatal(err)
			}

			// Make the request and check the response.
			tt.h.Service().ServeHTTP(rr, req)

			if rr.Code != tt.want {
				t.Errorf("expected status %d, got %d", tt.want, rr.Code)
			}
			if tt.want == http.StatusOK {
				pass, err := tt.h.AuthStorage.GetPass(tt.loginData.Login)
				if err != nil {
					t.Errorf("user not found in test storage: %v", err)
				}
				if pass != tt.loginData.Password {
					t.Errorf("wrong password")
				}
			}
		})
	}
}

func TestWebService_Login(t *testing.T) {
	key := "abcd"
	config.Cfg.Key = key
	mockAuthstorage := &auth.AuthMemStorage{
		Data: make(map[string]string),
	}
	mockstorage := &database.MemStorage{
		Data: make(map[string]string),
	}

	mockAuthstorage.Data["user123"] = "secretpass"

	var mockservice = &(WebService{
		Storage:     mockstorage,
		AuthStorage: mockAuthstorage,
		Mu:          &sync.Mutex{},
	})

	token123, err := auth.GenerateToken("user123")
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name      string
		loginData auth.LoginData
		token     string
		want      int
	}{
		{
			name: "Authorized request",
			loginData: auth.LoginData{
				Login:    "user123",
				Password: "secretpass",
			},
			token: token123,
			want:  http.StatusOK,
		},
		{
			name: "Wrong user",
			loginData: auth.LoginData{
				Login:    "unknownuser",
				Password: "verysecretpassword",
			},
			token: "",
			want:  http.StatusUnauthorized,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			//подготовка боди - json структуры в формате loginData
			rbody, err := json.Marshal(tt.loginData)
			if err != nil {
				t.Fatal(err)
			}

			rr := httptest.NewRecorder()
			req, err := http.NewRequest(http.MethodPost, "/api/user/login", bytes.NewBuffer(rbody))
			if err != nil {
				t.Fatal(err)
			}

			// Make the request and check the response.
			mockservice.Service().ServeHTTP(rr, req)

			if rr.Code != tt.want {
				t.Errorf("expected status %d, got %d", tt.want, rr.Code)
			}

		})
	}
}
