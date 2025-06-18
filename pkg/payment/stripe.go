package payment

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/iAmImran007/Code_War/pkg/database"
	"github.com/iAmImran007/Code_War/pkg/modles"
	"github.com/stripe/stripe-go/v76"
	"github.com/stripe/stripe-go/v76/checkout/session"
	"github.com/stripe/stripe-go/v76/customer"
	"github.com/stripe/stripe-go/v76/subscription"
	"github.com/stripe/stripe-go/v76/webhook"
)

type StripeService struct {
	DB *database.Databse
}

func NewStripeService(db *database.Databse) *StripeService {
	stripe.Key = os.Getenv("STRIPE_SECRET_KEY")
	return &StripeService{DB: db}
}

type CheckoutRequest struct {
	PlanType string `json:"plan_type"` // "monthly" or "yearly"
	UserID   uint   `json:"user_id"`
}

func (s *StripeService) CreateCheckoutSession(w http.ResponseWriter, r *http.Request) {
	var req CheckoutRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Get user
	var user modles.User
	if err := s.DB.Db.First(&user, req.UserID).Error; err != nil {
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}

	// Create or get Stripe customer
	customerID, err := s.getOrCreateCustomer(user)
	if err != nil {
		http.Error(w, "Failed to create customer", http.StatusInternalServerError)
		return
	}

	// Get price ID based on plan type
	var priceID string
	switch req.PlanType {
	case "monthly":
		priceID = os.Getenv("STRIPE_MONTHLY_PRICE_ID")
	case "yearly":
		priceID = os.Getenv("STRIPE_YEARLY_PRICE_ID")
	default:
		http.Error(w, "Invalid plan type", http.StatusBadRequest)
		return
	}

	params := &stripe.CheckoutSessionParams{
		Customer: stripe.String(customerID),
		PaymentMethodTypes: stripe.StringSlice([]string{
			"card",
		}),
		LineItems: []*stripe.CheckoutSessionLineItemParams{
			{
				Price:    stripe.String(priceID),
				Quantity: stripe.Int64(1),
			},
		},
		Mode:       stripe.String(string(stripe.CheckoutSessionModeSubscription)),
		SuccessURL: stripe.String(os.Getenv("DOMAIN") + "/success"),
		CancelURL:  stripe.String(os.Getenv("DOMAIN") + "/cancel"),
		Metadata: map[string]string{
			"user_id":   fmt.Sprintf("%d", req.UserID),
			"plan_type": req.PlanType,
		},
	}

	sess, err := session.New(params)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"checkout_url": sess.URL,
	})
}

func (s *StripeService) getOrCreateCustomer(user modles.User) (string, error) {
	// Check if customer already exists
	var sub modles.Subscription
	if err := s.DB.Db.Where("user_id = ?", user.ID).First(&sub).Error; err == nil {
		return sub.StripeCustomerID, nil
	}

	// Create new customer
	params := &stripe.CustomerParams{
		Email: stripe.String(user.Email),
		Metadata: map[string]string{
			"user_id": fmt.Sprintf("%d", user.ID),
		},
	}

	cust, err := customer.New(params)
	if err != nil {
		return "", err
	}

	return cust.ID, nil
}

// func (s *StripeService) HandleWebhook(w http.ResponseWriter, r *http.Request) {
// 	const MaxBodyBytes = int64(65536)
// 	r.Body = http.MaxBytesReader(w, r.Body, MaxBodyBytes)
	
// 	payload, err := io.ReadAll(r.Body)
// 	if err != nil {
// 		http.Error(w, err.Error(), http.StatusServiceUnavailable)
// 		return
// 	}

// 	event, err := webhook.ConstructEvent(payload, r.Header.Get("Stripe-Signature"), os.Getenv("STRIPE_WEBHOOK_SECRET"))
// 	if err != nil {
// 		http.Error(w, err.Error(), http.StatusBadRequest)
// 		return
// 	}

// 	switch event.Type {
// 	case "checkout.session.completed":
// 		var session stripe.CheckoutSession
// 		if err := json.Unmarshal(event.Data.Raw, &session); err != nil {
// 			log.Printf("Error parsing webhook JSON: %v", err)
// 			http.Error(w, err.Error(), http.StatusBadRequest)
// 			return
// 		}
// 		s.handleCheckoutCompleted(session)

// 	case "invoice.payment_succeeded":
// 		var invoice stripe.Invoice
// 		if err := json.Unmarshal(event.Data.Raw, &invoice); err != nil {
// 			log.Printf("Error parsing webhook JSON: %v", err)
// 			http.Error(w, err.Error(), http.StatusBadRequest)
// 			return
// 		}
// 		s.handlePaymentSucceeded(invoice)

// 	case "customer.subscription.deleted":
// 		var sub stripe.Subscription
// 		if err := json.Unmarshal(event.Data.Raw, &sub); err != nil {
// 			log.Printf("Error parsing webhook JSON: %v", err)
// 			http.Error(w, err.Error(), http.StatusBadRequest)
// 			return
// 		}
// 		s.handleSubscriptionCanceled(sub)
// 	}

// 	w.WriteHeader(http.StatusOK)
// }

func (s *StripeService) HandleWebhook(w http.ResponseWriter, r *http.Request) {
	fmt.Println("🔔 Webhook received!")
	fmt.Printf("📝 Request method: %s\n", r.Method)
	fmt.Printf("📝 Request URL: %s\n", r.URL.String())
	
	const MaxBodyBytes = int64(65536)
	r.Body = http.MaxBytesReader(w, r.Body, MaxBodyBytes)
	
	payload, err := io.ReadAll(r.Body)
	if err != nil {
		fmt.Printf("❌ Error reading body: %v\n", err)
		http.Error(w, err.Error(), http.StatusServiceUnavailable)
		return
	}
	fmt.Printf("✅ Body read successfully, length: %d bytes\n", len(payload))

	// Get signature and webhook secret
	signature := r.Header.Get("Stripe-Signature")
	webhookSecret := os.Getenv("STRIPE_WEBHOOK_SECRET")
	
	fmt.Printf("📝 Stripe signature: %s\n", signature)
	fmt.Printf("🔑 Webhook secret: %s\n", webhookSecret)
	
	if signature == "" {
		fmt.Println("❌ No Stripe-Signature header found!")
		http.Error(w, "Missing Stripe-Signature header", http.StatusBadRequest)
		return
	}
	
	if webhookSecret == "" {
		fmt.Println("❌ No webhook secret found in environment!")
		http.Error(w, "Missing webhook secret", http.StatusBadRequest)
		return
	}

	// Use ConstructEventWithOptions to ignore API version mismatch
	event, err := webhook.ConstructEventWithOptions(
		payload,
		signature,
		webhookSecret,
		webhook.ConstructEventOptions{
			IgnoreAPIVersionMismatch: true,
		},
	)
	if err != nil {
		fmt.Printf("❌ Signature verification failed: %v\n", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	fmt.Printf("✅ Signature verified! Event type: %s\n", event.Type)

	switch event.Type {
	case "checkout.session.completed":
		fmt.Println("🛒 Processing checkout.session.completed")
		var session stripe.CheckoutSession
		if err := json.Unmarshal(event.Data.Raw, &session); err != nil {
			fmt.Printf("❌ Error parsing checkout session JSON: %v\n", err)
			log.Printf("Error parsing webhook JSON: %v", err)
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		fmt.Printf("✅ Checkout session parsed: %s\n", session.ID)
		s.handleCheckoutCompleted(session)

	case "invoice.payment_succeeded":
		fmt.Println("💰 Processing invoice.payment_succeeded")
		var invoice stripe.Invoice
		if err := json.Unmarshal(event.Data.Raw, &invoice); err != nil {
			fmt.Printf("❌ Error parsing invoice JSON: %v\n", err)
			log.Printf("Error parsing webhook JSON: %v", err)
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		fmt.Printf("✅ Invoice parsed: %s\n", invoice.ID)
		s.handlePaymentSucceeded(invoice)

	case "customer.subscription.deleted":
		fmt.Println("🚫 Processing customer.subscription.deleted")
		var sub stripe.Subscription
		if err := json.Unmarshal(event.Data.Raw, &sub); err != nil {
			fmt.Printf("❌ Error parsing subscription JSON: %v\n", err)
			log.Printf("Error parsing webhook JSON: %v", err)
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		fmt.Printf("✅ Subscription parsed: %s\n", sub.ID)
		s.handleSubscriptionCanceled(sub)
		
	default:
		fmt.Printf("ℹ️ Unhandled event type: %s\n", event.Type)
	}

	fmt.Println("✅ Webhook processed successfully!")
	w.WriteHeader(http.StatusOK)
}

/*
func (s *StripeService) handleCheckoutCompleted(session stripe.CheckoutSession) {
	userID := session.Metadata["user_id"]
	planType := session.Metadata["plan_type"]

	// Get subscription details
	sub, err := subscription.Get(session.Subscription.ID, nil)
	if err != nil {
		log.Printf("Error getting subscription: %v", err)
		return
	}

	// Save subscription to database
	subscription := modles.Subscription{
		UserID:           parseUint(userID),
		StripeCustomerID: session.Customer.ID,
		SubscriptionID:   session.Subscription.ID,
		PlanType:         planType,
		Status:           "active",
		CurrentPeriodEnd: time.Unix(sub.CurrentPeriodEnd, 0),
	}

	s.DB.Db.Create(&subscription)
}
*/


func (s *StripeService) handleCheckoutCompleted(session stripe.CheckoutSession) error {
	// Validate required metadata
	userID, exists := session.Metadata["user_id"]
	if !exists || userID == "" {
		return fmt.Errorf("user_id not found in session metadata")
	}

	planType, exists := session.Metadata["plan_type"]
	if !exists || planType == "" {
		return fmt.Errorf("plan_type not found in session metadata")
	}

	// Check if subscription ID exists
	var subscriptionID string
	if session.Subscription != nil {
		subscriptionID = session.Subscription.ID
	} else {
		return fmt.Errorf("subscription is nil in checkout session")
	}

	// Check if customer ID exists
	var customerID string
	if session.Customer != nil {
		customerID = session.Customer.ID
	} else {
		return fmt.Errorf("customer is nil in checkout session")
	}

	// Get subscription details from Stripe
	sub, err := subscription.Get(subscriptionID, nil)
	if err != nil {
		return fmt.Errorf("error getting subscription from Stripe: %v", err)
	}

	// Parse user ID with error handling
	parsedUserID, err := strconv.ParseUint(userID, 10, 64)
	if err != nil {
		return fmt.Errorf("invalid user_id format: %v", err)
	}

	// Save subscription to database
	subscriptionModel := modles.Subscription{
		UserID:           uint(parsedUserID),
		StripeCustomerID: customerID,
		SubscriptionID:   subscriptionID,
		PlanType:         planType,
		Status:           "active",
		CurrentPeriodEnd: time.Unix(sub.CurrentPeriodEnd, 0),
	}

	// Create subscription in database with error handling
	if err := s.DB.Db.Create(&subscriptionModel).Error; err != nil {
		return fmt.Errorf("error saving subscription to database: %v", err)
	}

	log.Printf("Successfully created subscription for user %s: %s", userID, subscriptionID)
	return nil
}


func (s *StripeService) handleSubscriptionCanceled(sub stripe.Subscription) {
	var subscription modles.Subscription
	if err := s.DB.Db.Where("subscription_id = ?", sub.ID).First(&subscription).Error; err != nil {
		log.Printf("Subscription not found: %v", err)
		return
	}

	now := time.Now()
	subscription.Status = "canceled"
	subscription.CanceledAt = &now
	s.DB.Db.Save(&subscription)
}

// func parseUint(s string) uint {
// 	// Simple conversion - in production, add proper error handling
// 	if s == "" {
// 		return 0
// 	}
// 	// Convert string to uint (simplified)
// 	var result uint
// 	fmt.Sscanf(s, "%d", &result)
// 	return result
// }

func (s *StripeService) handlePaymentSucceeded(invoice stripe.Invoice) {
	// Check if this invoice is associated with a subscription
	if invoice.Subscription == nil {
		log.Printf("Invoice %s is not associated with a subscription, skipping", invoice.ID)
		return
	}

	// Check if subscription ID exists
	if invoice.Subscription.ID == "" {
		log.Printf("Invoice %s has empty subscription ID", invoice.ID)
		return
	}

	log.Printf("Processing payment success for subscription: %s", invoice.Subscription.ID)

	// Update subscription status
	var sub modles.Subscription
	if err := s.DB.Db.Where("subscription_id = ?", invoice.Subscription.ID).First(&sub).Error; err != nil {
		log.Printf("Subscription not found: %v", err)
		return
	}

	// Get updated subscription from Stripe
	stripeSub, err := subscription.Get(invoice.Subscription.ID, nil)
	if err != nil {
		log.Printf("Error getting subscription: %v", err)
		return
	}

	// Additional safety check
	if stripeSub == nil {
		log.Printf("Retrieved subscription is nil for ID: %s", invoice.Subscription.ID)
		return
	}

	sub.Status = "active"
	sub.CurrentPeriodEnd = time.Unix(stripeSub.CurrentPeriodEnd, 0)
	
	if err := s.DB.Db.Save(&sub).Error; err != nil {
		log.Printf("Error saving subscription: %v", err)
		return
	}

	log.Printf("Successfully updated subscription %s to active", invoice.Subscription.ID)
}
