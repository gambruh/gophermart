package auth

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gambruh/gophermart/internal/config"
	"github.com/gambruh/gophermart/internal/storage"
	"github.com/go-chi/chi/v5"
)

type TestService struct {
	Storage storage.Storage
}

func (ts *TestService) Service() http.Handler {
	handler := func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("OK"))
	}
	// Create a new router, add the AuthMiddleware and the mock handler.
	r := chi.NewRouter()
	r.Use(AuthMiddleware)
	r.Get("/test", handler)

	return r
}
func TestAuthMiddleware(t *testing.T) {
	key := "abcd"
	config.Cfg.Key = key
	mockstorage := storage.MemStorage{
		Data: make(map[string]string),
	}
	mockstorage.Data["user123"] = "secretpassword"
	var mockservice = &(TestService{Storage: &mockstorage})

	token123, err := GenerateToken("user123")
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name     string
		login    string
		password string
		token    string
		want     int
	}{
		{
			name:     "Authorized request",
			login:    "user123",
			password: "usualpass",
			token:    token123,
			want:     http.StatusOK,
		},
		{
			name:     "Wrong token",
			login:    "unknownuser",
			password: "verysecretpassword",
			token:    "mybrainiswashedup",
			want:     http.StatusUnauthorized,
		},
		{
			name:     "No token",
			login:    "unknownuser",
			password: "verysecretpassword",
			token:    "",
			want:     http.StatusUnauthorized,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			rr := httptest.NewRecorder()
			req, err := http.NewRequest(http.MethodGet, "/test", nil)
			if err != nil {
				t.Fatal(err)
			}

			cookie := http.Cookie{
				Name:  "gophermart-auth",
				Value: tt.token,
			}
			req.AddCookie(&cookie)

			// Make the request and check the response.
			mockservice.Service().ServeHTTP(rr, req)

			if rr.Code != tt.want {
				t.Errorf("expected status %d, got %d", tt.want, rr.Code)
			}
		})
	}
}
