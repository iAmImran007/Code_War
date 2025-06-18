package modles

import (
	"time"
	"gorm.io/gorm"
)

type Subscription struct {
	gorm.Model
	UserID           uint      `json:"user_id" db:"user_id"`
	StripeCustomerID string    `json:"stripe_customer_id" db:"stripe_customer_id"`
	SubscriptionID   string    `json:"subscription_id" db:"subscription_id"`
	PlanType         string    `json:"plan_type" db:"plan_type"` // "monthly" or "yearly"
	Status           string    `json:"status" db:"status"`       // "active", "canceled", "expired"
	CurrentPeriodEnd time.Time `json:"current_period_end" db:"current_period_end"`
	CanceledAt       *time.Time `json:"canceled_at" db:"canceled_at"`
}

type GameUsage struct {
	gorm.Model
	UserID    uint      `json:"user_id" db:"user_id"`
	Date      time.Time `json:"date" db:"date"`
	GamesUsed int       `json:"games_used" db:"games_used"`
}