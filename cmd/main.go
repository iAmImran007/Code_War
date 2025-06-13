package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/iAmImran007/Code_War/pkg/database"
	"github.com/iAmImran007/Code_War/pkg/routes"
	"github.com/gorilla/handlers"
)

func main() {
	database.LoadEnv()

	db := database.Databse{}
	database.ConectToDb(&db)

	// Insert dummy problem into db
	if err := database.InsertDummyProblem(&db); err != nil {
		log.Printf("Error inserting dummy problem: %v", err)
	}

	// Initialize router with all routes
	router := routes.NewRouter(&db)

	//Configure CORS
	corsOpts := handlers.CORS(
		handlers.AllowedOrigins([]string{
			"http://127.0.0.1:5500", //vs code liveserver
		}),
		handlers.AllowedMethods([]string{
			"GET", "POST", "PUT", "DELETE", "OPTIONS",
		}),
		handlers.AllowedHeaders([]string{
			"Content-Type", 
			"Authorization", 
			"X-Requested-With",
			"Accept",
			"Origin",
		}),
		handlers.AllowCredentials(), // for auth cookies/tokens
	)

	//bind the cors
	fmt.Println("Server running on localhost:8080")
	log.Fatal(http.ListenAndServe(":8080", corsOpts(router.Router)))
}
