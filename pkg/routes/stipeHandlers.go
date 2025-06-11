package routes

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/iAmImran007/Code_War/pkg/middleware"
)

func (r *Routes) handleGameWithLimit(w http.ResponseWriter, req *http.Request) {
	// Get user ID from middleware context
	userContext, ok := middleware.GetUserFromContext(req)
	if !ok {
		http.Error(w, "User not authenticated", http.StatusUnauthorized)
		return
	}
	userID := userContext.UserID

	// Check if user can play
	canPlay, err := r.GameLimit.CanPlayGame(userID)
	if err != nil {
		http.Error(w, "Error checking game limit", http.StatusInternalServerError)
		return
	}

	if !canPlay {
		http.Error(w, "Daily game limit reached. Upgrade to premium for unlimited games.", http.StatusForbidden)
		return
	}

	// Increment game usage
	if err := r.GameLimit.IncrementGameUsage(userID); err != nil {
		// Log error but don't block the game
		log.Printf("Error incrementing game usage: %v", err)
	}

	// Call original game handler
	r.GameRoom.HandleWs(w, req)
}

func (r *Routes) handleSubscriptionStatus(w http.ResponseWriter, req *http.Request) {
	userContext, ok := middleware.GetUserFromContext(req)
	if !ok {
		http.Error(w, "User not authenticated", http.StatusUnauthorized)
		return
	}
	userID := userContext.UserID

	subscription, err := r.GameLimit.GetSubscriptionStatus(userID)
	if err != nil {
		// User has no subscription (free user)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"is_premium": false,
			"plan_type":  "free",
			"status":     "free",
		})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"is_premium":         subscription.Status == "active",
		"plan_type":          subscription.PlanType,
		"status":             subscription.Status,
		"current_period_end": subscription.CurrentPeriodEnd,
	})
}
