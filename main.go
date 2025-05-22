package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/gorilla/mux"
)

func main() {
	LoadEnv()

	db := Databse{}
	ConectToDb(&db)

	// Insert dummy problem into db
	if err := InsertDummyProblem(&db); err != nil {
		log.Printf("Error inserting dummy problem: %v", err)
	}

	r := mux.NewRouter()

	rm := NewRoom(&db)

	r.HandleFunc("/ws", rm.HandleWs)

	fmt.Println("Server running on localhost:8080")
	log.Fatal(http.ListenAndServe(":8080", r))
}