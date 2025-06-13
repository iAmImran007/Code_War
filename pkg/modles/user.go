package modles

import (
	"time"

	//"github.com/golang-jwt/jwt/v5"
	"gorm.io/gorm"
)

type User struct {
	gorm.Model
	Email     string    `json:"email" db:"email"`
	Password  string    `json:"-" db:"password_hash"`
	Role      string    `gorm:"default:user" json:"role" db:"role"`
	//Rating    int       `json:"rating" gorm:"default:100"`
	//GamesWon  int       `json:"games_won" gorm:"default:0"`
	//GameLost  int       `json:"games_lost" gorm:"default:0"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
}

// type GameHistory struct{
// 	gorm.Model
// 	WinnerID   uint      `json:"winner_id" gorm:"not null"`
// 	LoserID    uint      `json:"loser_id" gorm:"not null"`
// 	ProblemID  uint      `json:"problem_id" gorm:"not null"`
// 	RatingGain int       `json:"rating_gain" gorm:"default:5"`
// 	CreatedAt time.Time `json:"created_at"`

// 	Winner  User `json:"winner" gorm:"foreignKey:WinnerID"`
// 	Loser   User `json:"loser" gorm:"foreignKey:LoserID"`
// }

type RefreshToken struct {
	gorm.Model
	UserID    uint      `json:"user_id" db:"user_id"`
	Token     string    `json:"token" db:"token"`
	ExpiresAt time.Time `json:"expires_at" db:"expires_at"`
	IsRevoked bool      `json:"is_revoked" db:"is_revoked"`
}
