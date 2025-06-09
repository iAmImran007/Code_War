package routes

import (
	"github.com/gorilla/mux"

	"github.com/iAmImran007/Code_War/pkg/database"
	"github.com/iAmImran007/Code_War/pkg/game"
	"github.com/iAmImran007/Code_War/pkg/middleware"
)

type Routes struct {
	Router         *mux.Router
	Db             *database.Databse
	AuthMiddleware *middleware.AuthMiddleware
	GameRoom       *game.Room
}

func NewRouter(db *database.Databse) *Routes {
	r := &Routes{
		Router:         mux.NewRouter(),
		Db:             db,
		AuthMiddleware: middleware.NewAuthMiddleware(db),
		GameRoom:       game.NewRoom(db),
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

	// Protected routes
	r.Router.HandleFunc("/logout", r.AuthMiddleware.RequireAuth(r.handleLogout)).Methods("POST")
	r.Router.HandleFunc("/profile/{id}", r.AuthMiddleware.RequireAuth(r.handleProfile)).Methods("GET")
	r.Router.HandleFunc("/ws", r.AuthMiddleware.RequireAuth(r.GameRoom.HandleWs))
}
