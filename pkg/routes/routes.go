package routes

import (
	"github.com/gorilla/mux"

	"github.com/iAmImran007/Code_War/pkg/database"
	"github.com/iAmImran007/Code_War/pkg/game"
	"github.com/iAmImran007/Code_War/pkg/middleware"
	"github.com/iAmImran007/Code_War/pkg/payment"
)

type Routes struct {
	Router         *mux.Router
	Db             *database.Databse
	AuthMiddleware *middleware.AuthMiddleware
	GameRoom       *game.Room
	StripieService *payment.StripeService
	GameLimit      *game.GameLimitService
}

func NewRouter(db *database.Databse) *Routes {
	r := &Routes{
		Router:         mux.NewRouter(),
		Db:             db,
		AuthMiddleware: middleware.NewAuthMiddleware(db),
		GameRoom:       game.NewRoom(db),
		StripieService: payment.NewStripeService(db),
		GameLimit:      game.NewGameLimitService(db),
	}

	r.setupRoutes()

	return r
}

func (r *Routes) setupRoutes() {
	// Public routes
	r.Router.HandleFunc("/home", r.handleHome).Methods("GET")
	r.Router.HandleFunc("/signup", r.handleSignUp).Methods("POST")
	r.Router.HandleFunc("/login", r.handleLogIn).Methods("POST")
	r.Router.HandleFunc("/refresh-token", r.handleRefreshToken).Methods("POST")
	r.Router.HandleFunc("/webhook", r.StripieService.HandleWebhook).Methods("POST")
	r.Router.HandleFunc("/problems", r.GetAllProblems).Methods("GET")
	r.Router.HandleFunc("/check-auth", r.handleCheckAuth).Methods("GET")

	//testig without auctantication
	//r.Router.HandleFunc("/submit/{id}", r.HandleSubmition).Methods("POST")
	//r.Router.HandleFunc("/ws", r.GameRoom.HandleWs)

	// Protected routes
	r.Router.HandleFunc("/logout", r.AuthMiddleware.RequireAuth(r.handleLogout)).Methods("POST")
	r.Router.HandleFunc("/profile/{id}", r.AuthMiddleware.RequireAuth(r.handleProfile)).Methods("GET")

	//r.Router.HandleFunc("/ws", r.AuthMiddleware.RequireAuth(r.GameRoom.HandleWs))
	r.Router.HandleFunc("/ws", r.AuthMiddleware.RequireAuth(r.handleGameWithLimit))
	r.Router.HandleFunc("/problem/{id}", r.AuthMiddleware.RequireAuth(r.GetProblemById)).Methods("GET")
	r.Router.HandleFunc("/submit/{id}", r.AuthMiddleware.RequireAuth(r.HandleSubmition)).Methods("POST")

	//stripe routes
	// Stripe routes
	r.Router.HandleFunc("/create-checkout-session", r.AuthMiddleware.RequireAuth(r.StripieService.CreateCheckoutSession)).Methods("POST")
	r.Router.HandleFunc("/webhook", r.StripieService.HandleWebhook).Methods("POST")
	r.Router.HandleFunc("/subscription-status", r.AuthMiddleware.RequireAuth(r.handleSubscriptionStatus)).Methods("GET")

}
