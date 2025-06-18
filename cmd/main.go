package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/gorilla/handlers"
	"github.com/iAmImran007/Code_War/pkg/database"
	"github.com/iAmImran007/Code_War/pkg/routes"
)

func main() {
	if err := database.LoadEnv(); err != nil {
		log.Fatalf("Failed to load environment variables: %v", err)
	}

	db := database.Databse{}
	if err := database.ConectToDb(&db); err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

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
			"http://localhost:8080",
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
