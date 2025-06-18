package routes

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gorilla/mux"
	"github.com/iAmImran007/Code_War/pkg/auth"
	"github.com/iAmImran007/Code_War/pkg/modles"
	"github.com/iAmImran007/Code_War/pkg/utils"
)

type SignUpRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type Response struct {
	Success bool        `json:"success"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
	Errors  []string    `json:"errors,omitempty"`
}

func (r *Routes) handleSignUp(w http.ResponseWriter, req *http.Request) {
	// Set security headers
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("X-Frame-Options", "DENY")
	w.Header().Set("X-XSS-Protection", "1; mode=block")

	// Check content type
	if !strings.Contains(req.Header.Get("Content-Type"), "application/json") {
		w.WriteHeader(http.StatusUnsupportedMediaType)
		json.NewEncoder(w).Encode(Response{
			Success: false,
			Message: "Content-Type must be application/json",
		})
		return
	}

	// Limit request body size (1MB)
	req.Body = http.MaxBytesReader(w, req.Body, 1048576)

	// Read and decode request body
	var signUpReq SignUpRequest
	decoder := json.NewDecoder(req.Body)
	decoder.DisallowUnknownFields() // Security: reject unknown fields

	if err := decoder.Decode(&signUpReq); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(Response{
			Success: false,
			Message: "Invalid request body",
		})
		return
	}

	// Sanitize inputs
	signUpReq.Email = utils.SanitizeString(signUpReq.Email)
	signUpReq.Password = utils.SanitizeString(signUpReq.Password)

	// Validate inputs with detailed error messages
	var errors []string

	if !utils.ValidateEmail(signUpReq.Email) {
		errors = append(errors, "Valid email address is required")
	}

	if !utils.ValidatePassword(signUpReq.Password) {
		errors = append(errors, "Password must be 8-128 characters with at least 3 of: uppercase, lowercase, number, special character")
	}

	if utils.IsWeakPassword(signUpReq.Password) {
		errors = append(errors, "Password is too common, please choose a stronger password")
	}

	if len(errors) > 0 {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(Response{
			Success: false,
			Message: "Validation failed",
			Errors:  errors,
		})
		return
	}

	// Normalize email to lowercase
	signUpReq.Email = strings.ToLower(signUpReq.Email)

	// Check if user already exists
	var existingUser modles.User
	if err := r.Db.Db.Where("email = ?", signUpReq.Email).First(&existingUser).Error; err == nil {
		w.WriteHeader(http.StatusConflict)
		json.NewEncoder(w).Encode(Response{
			Success: false,
			Message: "User already exists",
		})
		return
	}

	// Hash password with high cost for security
	hashPassword, err := utils.HashPassword(signUpReq.Password)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(Response{
			Success: false,
			Message: "Failed to process password",
		})
		return
	}

	// Create new user with default role
	user := modles.User{
		Email:    signUpReq.Email,
		Password: hashPassword,
		Role:     "user", // Always default to user role
	}

	// Save user to database
	if err := r.Db.Db.Create(&user).Error; err != nil {
		fmt.Printf("Error creating user: %v\n", err)
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(Response{
			Success: false,
			Message: "Failed to create user",
		})
		return
	}

	fmt.Printf("User created with ID: %d, Email: %s\n", user.ID, user.Email)

	// Generate JWT tokens
	tokenPair, err := auth.GanaretTokenPair(user.ID, user.Email, user.Role)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(Response{
			Success: false,
			Message: "Failed to generate authentication tokens",
		})
		return
	}

	// Store refresh token in database
	refreshToken := modles.RefreshToken{
		UserID:    user.ID,
		Token:     tokenPair.RefreshToken,
		ExpiresAt: time.Now().Add(7 * 24 * time.Hour),
		IsRevoked: false,
	}

	if err := r.Db.Db.Create(&refreshToken).Error; err != nil {
		// Log error but don't fail the request
		// User is created successfully, they can login again
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(Response{
			Success: true,
			Message: "User created successfully, please login",
			Data: map[string]interface{}{
				"user_id": user.ID,
				"email":   user.Email,
			},
		})
		return
	}

	// Determine if we're in development mode
	isDevelopment := os.Getenv("ENVIRONMENT") == "development"

	// Set secure HTTP-only cookies
	http.SetCookie(w, &http.Cookie{
		Name:     "access_token",
		Value:    tokenPair.AccessToken,
		HttpOnly: true,
		Secure:   !isDevelopment, // false for development, true for production
		SameSite: http.SameSiteStrictMode,
		MaxAge:   15 * 60, // 15 minutes
		Path:     "/",
	})

	http.SetCookie(w, &http.Cookie{
		Name:     "refresh_token",
		Value:    tokenPair.RefreshToken,
		HttpOnly: true,
		Secure:   !isDevelopment, // false for development, true for production
		SameSite: http.SameSiteStrictMode,
		MaxAge:   7 * 24 * 60 * 60, // 7 days
		Path:     "/",
	})
	fmt.Println("User created successfully")
	// Success response (don't include sensitive data)
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(Response{
		Success: true,
		Message: "User created successfully",
		Data: map[string]interface{}{
			"user_id": user.ID,
			"email":   user.Email,
			"role":    user.Role,
		},
	})
}

func (r *Routes) handleLogIn(w http.ResponseWriter, req *http.Request) {
	// Set security headers
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("X-Frame-Options", "DENY")
	w.Header().Set("X-XSS-Protection", "1; mode=block")

	// Check content type
	if !strings.Contains(req.Header.Get("Content-Type"), "application/json") {
		w.WriteHeader(http.StatusUnsupportedMediaType)
		json.NewEncoder(w).Encode(Response{
			Success: false,
			Message: "Content-Type must be application/json",
		})
		return
	}

	// Limit request body size (1MB)
	req.Body = http.MaxBytesReader(w, req.Body, 1048576)

	// Read and decode request body
	var loginReq LoginRequest
	decoder := json.NewDecoder(req.Body)
	decoder.DisallowUnknownFields() // Security: reject unknown fields

	if err := decoder.Decode(&loginReq); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(Response{
			Success: false,
			Message: "Invalid request body",
		})
		return
	}

	// Sanitize inputs
	loginReq.Email = utils.SanitizeString(loginReq.Email)
	loginReq.Password = utils.SanitizeString(loginReq.Password)

	// Validate inputs with detailed error messages
	var errors []string

	if !utils.ValidateEmail(loginReq.Email) {
		errors = append(errors, "Valid email address is required")
	}

	if !utils.ValidatePassword(loginReq.Password) {
		errors = append(errors, "Valid password is required")
	}

	if len(errors) > 0 {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(Response{
			Success: false,
			Message: "Validation failed",
			Errors:  errors,
		})
		return
	}

	// Normalize email to lowercase
	loginReq.Email = strings.ToLower(loginReq.Email)

	// Find user
	var user modles.User
	if err := r.Db.Db.Where("email = ?", loginReq.Email).First(&user).Error; err != nil {
		fmt.Printf("Login attempt failed for email %s: %v\n", loginReq.Email, err)
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(Response{
			Success: false,
			Message: "Invalid credentials",
		})
		return
	}

	fmt.Printf("User found with ID: %d, Email: %s\n", user.ID, user.Email)

	// Verify password
	if err := utils.ComparePassword(user.Password, loginReq.Password); err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(Response{
			Success: false,
			Message: "Invalid credentials",
		})
		return
	}

	// Generate JWT tokens
	tokenPair, err := auth.GanaretTokenPair(user.ID, user.Email, user.Role)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(Response{
			Success: false,
			Message: "Failed to generate authentication tokens",
		})
		return
	}

	// Store refresh token in database
	refreshToken := modles.RefreshToken{
		UserID:    user.ID,
		Token:     tokenPair.RefreshToken,
		ExpiresAt: time.Now().Add(7 * 24 * time.Hour),
		IsRevoked: false,
	}

	if err := r.Db.Db.Create(&refreshToken).Error; err != nil {
		// Log error but don't fail the request
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(Response{
			Success: true,
			Message: "Login successful, please login again if tokens not set",
			Data: map[string]interface{}{
				"user_id": user.ID,
				"email":   user.Email,
			},
		})
		return
	}

	// Determine if we're in development mode
	isDevelopment := os.Getenv("ENVIRONMENT") == "development"

	// Set secure HTTP-only cookies
	http.SetCookie(w, &http.Cookie{
		Name:     "access_token",
		Value:    tokenPair.AccessToken,
		HttpOnly: true,
		Secure:   !isDevelopment, // false for development, true for production
		SameSite: http.SameSiteStrictMode,
		MaxAge:   15 * 60, // 15 minutes
		Path:     "/",
	})

	http.SetCookie(w, &http.Cookie{
		Name:     "refresh_token",
		Value:    tokenPair.RefreshToken,
		HttpOnly: true,
		Secure:   !isDevelopment, // false for development, true for production
		SameSite: http.SameSiteStrictMode,
		MaxAge:   7 * 24 * 60 * 60, // 7 days
		Path:     "/",
	})

	// Success response (don't include sensitive data)
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(Response{
		Success: true,
		Message: "Login successful",
		Data: map[string]interface{}{
			"user_id": user.ID,
			"email":   user.Email,
			"role":    user.Role,
		},
	})
}

func (r *Routes) handleLogout(w http.ResponseWriter, req *http.Request) {
	// Set security headers
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("X-Frame-Options", "DENY")
	w.Header().Set("X-XSS-Protection", "1; mode=block")

	// Only allow POST method for logout (security best practice)
	if req.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		json.NewEncoder(w).Encode(Response{
			Success: false,
			Message: "Method not allowed",
		})
		return
	}

	// Get refresh token from cookie
	refreshCookie, err := req.Cookie("refresh_token")
	if err == nil && refreshCookie.Value != "" {
		// Revoke refresh token in database
		if err := r.Db.Db.Model(&modles.RefreshToken{}).Where("token = ?", refreshCookie.Value).Update("is_revoked", true).Error; err != nil {
			// Log error but don't fail the logout process
			// Logout should always succeed from user perspective
		}
	}

	// Determine if we're in development mode
	isDevelopment := os.Getenv("ENVIRONMENT") == "development"

	// Clear access token cookie
	http.SetCookie(w, &http.Cookie{
		Name:     "access_token",
		Value:    "",
		HttpOnly: true,
		Secure:   !isDevelopment, // false for development, true for production
		SameSite: http.SameSiteStrictMode,
		MaxAge:   -1, // Expire immediately
		Path:     "/",
	})

	// Clear refresh token cookie
	http.SetCookie(w, &http.Cookie{
		Name:     "refresh_token",
		Value:    "",
		HttpOnly: true,
		Secure:   !isDevelopment, // false for development, true for production
		SameSite: http.SameSiteStrictMode,
		MaxAge:   -1, // Expire immediately
		Path:     "/",
	})

	// Success response (consistent with other handlers)
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(Response{
		Success: true,
		Message: "Logout successful",
		Data:    nil,
	})
}

func (r *Routes) handleRefreshToken(w http.ResponseWriter, req *http.Request) {
	// Set security headers
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("X-Frame-Options", "DENY")
	w.Header().Set("X-XSS-Protection", "1; mode=block")

	// Only allow POST method for token refresh (security best practice)
	if req.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		json.NewEncoder(w).Encode(Response{
			Success: false,
			Message: "Method not allowed",
		})
		return
	}

	// Get refresh token from cookie
	refreshCookie, err := req.Cookie("refresh_token")
	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(Response{
			Success: false,
			Message: "Refresh token not found",
		})
		return
	}

	// Check if cookie has value
	if refreshCookie.Value == "" {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(Response{
			Success: false,
			Message: "Invalid refresh token",
		})
		return
	}

	// Validate refresh token format and signature
	claims, err := auth.ValidateToken(refreshCookie.Value)
	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(Response{
			Success: false,
			Message: "Invalid refresh token",
		})
		return
	}

	// Check if refresh token exists in database and is not revoked
	var refreshToken modles.RefreshToken
	if err := r.Db.Db.Where("token = ? AND is_revoked = false AND expires_at > ?",
		refreshCookie.Value, time.Now()).First(&refreshToken).Error; err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(Response{
			Success: false,
			Message: "Invalid or expired refresh token",
		})
		return
	}

	// Verify token belongs to the same user (security check)
	if refreshToken.UserID != claims.UserID {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(Response{
			Success: false,
			Message: "Invalid refresh token",
		})
		return
	}

	// Revoke old refresh token (security: one-time use)
	if err := r.Db.Db.Model(&refreshToken).Update("is_revoked", true).Error; err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(Response{
			Success: false,
			Message: "Failed to process token refresh",
		})
		return
	}

	// Generate new token pair
	tokenPair, err := auth.GanaretTokenPair(claims.UserID, claims.Email, claims.Role)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(Response{
			Success: false,
			Message: "Failed to generate authentication tokens",
		})
		return
	}

	// Store new refresh token in database
	newRefreshToken := modles.RefreshToken{
		UserID:    claims.UserID,
		Token:     tokenPair.RefreshToken,
		ExpiresAt: time.Now().Add(7 * 24 * time.Hour),
		IsRevoked: false,
	}

	if err := r.Db.Db.Create(&newRefreshToken).Error; err != nil {
		// If we can't store the new refresh token, the operation fails
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(Response{
			Success: false,
			Message: "Failed to store refresh token",
		})
		return
	}

	// Determine if we're in development mode
	isDevelopment := os.Getenv("ENVIRONMENT") == "development"

	// Set new secure HTTP-only cookies
	http.SetCookie(w, &http.Cookie{
		Name:     "access_token",
		Value:    tokenPair.AccessToken,
		HttpOnly: true,
		Secure:   !isDevelopment, // false for development, true for production
		SameSite: http.SameSiteStrictMode,
		MaxAge:   15 * 60, // 15 minutes
		Path:     "/",
	})

	http.SetCookie(w, &http.Cookie{
		Name:     "refresh_token",
		Value:    tokenPair.RefreshToken,
		HttpOnly: true,
		Secure:   !isDevelopment, // false for development, true for production
		SameSite: http.SameSiteStrictMode,
		MaxAge:   7 * 24 * 60 * 60, // 7 days
		Path:     "/",
	})

	// Success response (consistent with other handlers)
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(Response{
		Success: true,
		Message: "Tokens refreshed successfully",
		Data: map[string]interface{}{
			"user_id": claims.UserID,
			"email":   claims.Email,
			"role":    claims.Role,
		},
	})
}

func (r *Routes) handleProfile(w http.ResponseWriter, req *http.Request) {
	// Set security headers
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("X-Frame-Options", "DENY")
	w.Header().Set("X-XSS-Protection", "1; mode=block")

	// Only allow GET method
	if req.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		json.NewEncoder(w).Encode(Response{
			Success: false,
			Message: "Method not allowed",
		})
		return
	}

	// Get user ID from URL path
	vars := mux.Vars(req)
	userID := vars["id"]

	if userID == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(Response{
			Success: false,
			Message: "User ID is required",
		})
		return
	}

	// Find user by ID
	var user modles.User
	if err := r.Db.Db.Where("id = ?", userID).First(&user).Error; err != nil {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(Response{
			Success: false,
			Message: "User not found",
		})
		return
	}

	// Success response with user profile data
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(Response{
		Success: true,
		Message: "Profile retrieved successfully",
		Data: map[string]interface{}{
			"user_id": user.ID,
			"email":   user.Email,
			"role":    user.Role,
			"rating": user.Rating,
			"solved_problems": user.SolvedProblems,
		},
	})
}

func (r *Routes) handleHome(w http.ResponseWriter, req *http.Request) {
	// Set security headers
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("X-Frame-Options", "DENY")
	w.Header().Set("X-XSS-Protection", "1; mode=block")

	// Only allow GET method
	if req.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		json.NewEncoder(w).Encode(Response{
			Success: false,
			Message: "Method not allowed",
		})
		return
	}

	// Success response
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(Response{
		Success: true,
		Message: "Welcome to Code War API",
		Data: map[string]interface{}{
			"name":        "Code War API",
			"version":     "1.0.0",
			"description": "A competitive coding platform API",
		},
	})
}

/*
func (r *Routes) handleCheckAuth(w http.ResponseWriter, req *http.Request) {
	// Set security headers
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("X-Frame-Options", "DENY")
	w.Header().Set("X-XSS-Protection", "1; mode=block")

	// Only allow GET method
	if req.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		json.NewEncoder(w).Encode(Response{
			Success: false,
			Message: "Method not allowed",
		})
		return
	}

	// Get access token from cookie
	accessCookie, err := req.Cookie("access_token")
	if err != nil || accessCookie.Value == "" {
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(Response{
			Success: false,
			Message: "Not authenticated",
		})
		return
	}

	// Validate the token
	claims, err := auth.ValidateToken(accessCookie.Value)
	if err != nil {
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(Response{
			Success: false,
			Message: "Invalid token",
		})
		return
	}

	// Success response
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(Response{
		Success: true,
		Message: "Authenticated",
		Data: map[string]interface{}{
			"user_id": claims.UserID,
			"email":   claims.Email,
			"role":    claims.Role,
		},
	})
}
*/