package modles

import (
	"time"

	//"github.com/golang-jwt/jwt/v5"
	"gorm.io/gorm"
)


type User struct{
	gorm.Model
	Email       string    `json:"email" db:"email"`
	Password    string    `json:"-" db:"password_hash"` 
	Role        string     `gorm:"default:user" json:"role" db:"role"`
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time `json:"updated_at" db:"updated_at"`
}

type RefreshToken struct {
	gorm.Model
	UserID    uint      `json:"user_id" db:"user_id"`
	Token     string    `json:"token" db:"token"`
	ExpiresAt time.Time `json:"expires_at" db:"expires_at"`
	IsRevoked bool      `json:"is_revoked" db:"is_revoked"`
}


