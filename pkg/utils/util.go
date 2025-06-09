package utils

import (
	"regexp"
	"strings"
	"unicode"
	"golang.org/x/crypto/bcrypt"
)

func HashPassword(password string) (string, error) {
	hashByte, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(hashByte), nil
}

func ComparePassword(hashedPassword, password string) error {
	return bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(password))
}

// Enhanced email validation with regex
func ValidateEmail(email string) bool {
	if email == "" {
		return false
	}
	
	// Basic email regex pattern
	emailRegex := `^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`
	re := regexp.MustCompile(emailRegex)
	
	// Check length and format
	return len(email) <= 254 && re.MatchString(email)
}

// Strong password validation
func ValidatePassword(password string) bool {
	if len(password) < 8 || len(password) > 69 {
		return false
	}
	
	var (
		hasUpper   = false
		hasLower   = false
		hasNumber  = false
		hasSpecial = false
	)
	
	for _, char := range password {
		switch {
		case unicode.IsUpper(char):
			hasUpper = true
		case unicode.IsLower(char):
			hasLower = true
		case unicode.IsNumber(char):
			hasNumber = true
		case unicode.IsPunct(char) || unicode.IsSymbol(char):
			hasSpecial = true
		}
	}
	
	// Require at least 3 of 4 character types
	count := 0
	if hasUpper { count++ }
	if hasLower { count++ }
	if hasNumber { count++ }
	if hasSpecial { count++ }
	
	return count >= 3
}

// Sanitize input strings
func SanitizeString(input string) string {
	// Remove leading/trailing whitespace
	input = strings.TrimSpace(input)
	
	// Remove null bytes and control characters
	result := strings.Builder{}
	for _, r := range input {
		if r >= 32 && r != 127 { // Printable ASCII characters
			result.WriteRune(r)
		}
	}
	
	return result.String()
}

// Check for common weak passwords
func IsWeakPassword(password string) bool {
	weakPasswords := []string{
		"password", "123456", "password123", "admin", "qwerty",
		"letmein", "welcome", "monkey", "1234567890", "abc123",
	}
	
	lowerPassword := strings.ToLower(password)
	for _, weak := range weakPasswords {
		if lowerPassword == weak {
			return true
		}
	}
	
	return false
}