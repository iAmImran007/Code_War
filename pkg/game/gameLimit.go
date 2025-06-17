package game


import (
	"time"

	"github.com/iAmImran007/Code_War/pkg/database"
	"github.com/iAmImran007/Code_War/pkg/modles"
)

type GameLimitService struct {
	DB *database.Databse
}

func NewGameLimitService(db *database.Databse) *GameLimitService {
	return &GameLimitService{DB: db}
}

func (g *GameLimitService) CanPlayGame(userID uint) (bool, error) {
	// Check if user has active subscription
	if g.hasActiveSubscription(userID) {
		return true, nil // Premium users have unlimited games
	}

	// Check daily game usage for free users
	today := time.Now().Truncate(24 * time.Hour)
	var usage modles.GameUsage
	
	err := g.DB.Db.Where("user_id = ? AND date = ?", userID, today).First(&usage).Error
	if err != nil {
		// No usage record for today, user can play
		return true, nil
	}

	// Free users get 1 game per day
	return usage.GamesUsed < 10, nil
}

func (g *GameLimitService) IncrementGameUsage(userID uint) error {
	// Don't track usage for premium users
	if g.hasActiveSubscription(userID) {
		return nil
	}

	today := time.Now().Truncate(24 * time.Hour)
	var usage modles.GameUsage

	err := g.DB.Db.Where("user_id = ? AND date = ?", userID, today).First(&usage).Error
	if err != nil {
		// Create new usage record
		usage = modles.GameUsage{
			UserID:    userID,
			Date:      today,
			GamesUsed: 1,
		}
		return g.DB.Db.Create(&usage).Error
	}

	// Increment existing usage
	usage.GamesUsed++
	return g.DB.Db.Save(&usage).Error
}

func (g *GameLimitService) hasActiveSubscription(userID uint) bool {
	var subscription modles.Subscription
	err := g.DB.Db.Where("user_id = ? AND status = ? AND current_period_end > ?", 
		userID, "active", time.Now()).First(&subscription).Error
	return err == nil
}

func (g *GameLimitService) GetSubscriptionStatus(userID uint) (*modles.Subscription, error) {
	var subscription modles.Subscription
	err := g.DB.Db.Where("user_id = ?", userID).First(&subscription).Error
	if err != nil {
		return nil, err
	}
	return &subscription, nil
}