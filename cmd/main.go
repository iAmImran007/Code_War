package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/iAmImran007/Code_War/pkg/database"
	"github.com/iAmImran007/Code_War/pkg/routes"
	
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


	//bind the cors
	fmt.Println("Server running on localhost:8080")
	log.Fatal(http.ListenAndServe(":8080", router.Router))
}
