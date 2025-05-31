package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/iAmImran007/game-service/pkg/database"
	"github.com/iAmImran007/game-service/pkg/game"
)

func main() {
	database.LoadEnv()

	db := database.Databse{}
	database.ConectToDb(&db)

	// Insert dummy problem into db
	if err := database.InsertDummyProblem(&db); err != nil {
		log.Printf("Error inserting dummy problem: %v", err)
	}

	r := mux.NewRouter()

	rm := game.NewRoom(&db)

	r.HandleFunc("/ws", rm.HandleWs)

	fmt.Println("Server running on localhost:8080")
	log.Fatal(http.ListenAndServe(":8080", r))
}
