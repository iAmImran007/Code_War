package main

import (
	"fmt"
	"log"
	"math/rand"
	"os"
	"time"

	"github.com/joho/godotenv"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

type Databse struct {
	Db *gorm.DB
}

func LoadEnv() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error while loading the env files")
	}
}

func ConectToDb(db *Databse) {
	LoadEnv()

	dsn := fmt.Sprintf(
		"host=%s user=%s password=%s dbname=%s port=%s sslmode=%s",
		os.Getenv("DB_HOST"),
		os.Getenv("DB_USER"),
		os.Getenv("DB_PASSWORD"),
		os.Getenv("DB_NAME"),
		os.Getenv("DB_PORT"),
		os.Getenv("SSL_MODE"),
	)

	conn, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatalf("Failed to connect to the database: %v", err)
	}

	db.Db = conn

	err = db.Db.AutoMigrate(&ProblemPropaty{}, &TestCaesPropaty{})
	if err != nil {
		log.Printf("Failed to auto migrate the database: %v", err)
	} else {
		fmt.Println("Database auto migrate successfully")
	}
}

func GetDb(d *Databse) *gorm.DB {
	return d.Db
}

func GetRandomProblem(db *Databse) (*ProblemPropaty, error) {
	var problems []ProblemPropaty
	if err := db.Db.Preload("TestCases").Find(&problems).Error; err != nil {
		return nil, err
	}
	if len(problems) == 0 {
		return nil, fmt.Errorf("no problem found in database")
	}

	rand.Seed(time.Now().UnixNano())
	randomIndex := rand.Intn(len(problems))
	return &problems[randomIndex], nil
}

func InsertDummyProblem(db *Databse) error {
	// Check if problems already exist
	var count int64
	db.Db.Model(&ProblemPropaty{}).Count(&count)
	if count > 0 {
		fmt.Println("Problems already exist in database")
		return nil
	}

	// Create problems array
	problems := []ProblemPropaty{
		{
			Title:       "Two Sum",
			Description: "Given an array of integers nums and an integer target, return indices of the two numbers such that they add up to target.",
			HaderFile: `#include <iostream>
#include <vector>
using namespace std;`,
			FuncBody: `vector<int> twoSum(vector<int>& nums, int target) {
    // Your code here
}`,
			MainFunc: `int main() {
    int n, target;
    cin >> n;
    vector<int> nums(n);
    for(int i = 0; i < n; i++) {
        cin >> nums[i];
    }
    cin >> target;
    
    vector<int> result = twoSum(nums, target);
    cout << result[0] << " " << result[1] << endl;
    
    return 0;
}`,
			TestCases: []TestCaesPropaty{
				{
					Input:          "4\n2 7 11 15\n9",
					ExpectedOutput: "0 1",
				},
				{
					Input:          "3\n3 2 4\n6",
					ExpectedOutput: "1 2",
				},
				{
					Input:          "2\n3 3\n6",
					ExpectedOutput: "0 1",
				},
			},
		},
		{
			Title:       "Reverse Integer",
			Description: "Given a signed 32-bit integer x, return x with its digits reversed. If reversing x causes the value to go outside the signed 32-bit integer range, then return 0.",
			HaderFile: `#include <iostream>
#include <climits>
using namespace std;`,
			FuncBody: `int reverse(int x) {
    // Your code here
}`,
			MainFunc: `int main() {
    int x;
    cin >> x;
    
    int result = reverse(x);
    cout << result << endl;
    
    return 0;
}`,
			TestCases: []TestCaesPropaty{
				{
					Input:          "123",
					ExpectedOutput: "321",
				},
				{
					Input:          "-123",
					ExpectedOutput: "-321",
				},
				{
					Input:          "120",
					ExpectedOutput: "21",
				},
				{
					Input:          "0",
					ExpectedOutput: "0",
				},
			},
		},
	}

	// Insert all problems
	for _, problem := range problems {
		if err := db.Db.Create(&problem).Error; err != nil {
			return fmt.Errorf("failed to insert problem '%s': %v", problem.Title, err)
		}
		fmt.Printf("Problem '%s' inserted successfully\n", problem.Title)
	}

	return nil
}