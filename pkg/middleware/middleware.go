package middleware

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/iAmImran007/Code_War/pkg/auth"
	"github.com/iAmImran007/Code_War/pkg/database"
	"github.com/iAmImran007/Code_War/pkg/modles"
)

type contextKey string

const UserContextKey contextKey = "user"

type AuthMiddleware struct {
	Db *database.Databse
}

type Response struct {
	Success bool        `json:"success"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
	Errors  []string    `json:"errors,omitempty"`
}

type UserContext struct {
	UserID uint   `json:"user_id"`
	Email  string `json:"email"`
	Role   string `json:"role"`
}

func NewAuthMiddleware(db *database.Databse) *AuthMiddleware {
	return &AuthMiddleware{
		Db: db,
	}
}

func (am *AuthMiddleware) RequireAuth(next http.HandlerFunc) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Set security headers (consistent with your handlers)
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("X-XSS-Protection", "1; mode=block")

		// Get access token from cookie
		accessCookie, err := r.Cookie("access_token")
		if err != nil {
			w.WriteHeader(http.StatusUnauthorized)
			json.NewEncoder(w).Encode(Response{
				Success: false,
				Message: "Access token not found",
			})
			return
		}

		// Check if cookie has value
		if accessCookie.Value == "" {
			w.WriteHeader(http.StatusUnauthorized)
			json.NewEncoder(w).Encode(Response{
				Success: false,
				Message: "Invalid access token",
			})
			return
		}

		// Validate access token
		claims, err := auth.ValidateToken(accessCookie.Value)
		if err != nil {
			w.WriteHeader(http.StatusUnauthorized)
			json.NewEncoder(w).Encode(Response{
				Success: false,
				Message: "Invalid or expired access token",
			})
			return
		}

		// Verify user still exists in database
		var user modles.User
		if err := am.Db.Db.Where("id = ?", claims.UserID).First(&user).Error; err != nil {
			w.WriteHeader(http.StatusUnauthorized)
			json.NewEncoder(w).Encode(Response{
				Success: false,
				Message: "User not found",
			})
			return
		}

		// Add user context to request
		userContext := UserContext{
			UserID: claims.UserID,
			Email:  claims.Email,
			Role:   claims.Role,
		}

		ctx := context.WithValue(r.Context(), UserContextKey, userContext)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// Helper function to get user from context (for use in handlers)
func GetUserFromContext(r *http.Request) (*UserContext, bool) {
	user, ok := r.Context().Value(UserContextKey).(UserContext)
	return &user, ok
}