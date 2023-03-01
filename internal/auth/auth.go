package auth

import (
	"context"
	"database/sql"
	"net/http"
)

type UserID string

var CheckUsernameQuery = `
	"SELECT user_id FROM auth_tokens WHERE token = $1"

`

func NewUser() error {
	return nil
}

func AuthMiddleware(db *sql.DB) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			token := r.Header.Get("Authorization")
			if token == "" {
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}

			// Query the database to check if the token is valid
			var userID int
			err := db.QueryRow("SELECT user_id FROM auth_tokens WHERE token = $1", token).Scan(&userID)
			if err != nil {
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}

			// Attach the user ID to the request context

			ctx := context.WithValue(r.Context(), UserID("UserID"), userID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func protectedHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID := ctx.Value("userID").(int)

	// Check if user is logged in
	if userID == 0 {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	// Handle request for authenticated user
	// ...
}
