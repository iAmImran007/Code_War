package database

import (
	"fmt"
	"math/rand"
	"os"
	"time"

	"github.com/iAmImran007/Code_War/pkg/modles"
	"github.com/joho/godotenv"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

type Databse struct {
	Db    *gorm.DB
	Cache *Redis
}

func LoadEnv() error {
	err := godotenv.Load()
	if err != nil {
		return fmt.Errorf("error while loading the env files: %v", err)
	}
	return nil
}

func ConectToDb(db *Databse) error {
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
		return fmt.Errorf("failed to connect to the database: %v", err)
	}

	// Test the connection
	sqlDB, err := conn.DB()
	if err != nil {
		return fmt.Errorf("failed to get database instance: %v", err)
	}

	if err := sqlDB.Ping(); err != nil {
		return fmt.Errorf("failed to ping database: %v", err)
	}

	db.Db = conn

	// Auto migrate the schema
	err = db.Db.AutoMigrate(&modles.ProblemPropaty{}, &modles.TestCaesPropaty{}, &modles.User{}, &modles.RefreshToken{}, &modles.Subscription{}, &modles.GameUsage{}, &modles.Example{})
	if err != nil {
		return fmt.Errorf("failed to auto migrate the database: %v", err)
	}
	fmt.Println("Database auto migrate successfully")

	// Initialize Redis cache
	db.Cache = NewServerChace(db)
	if db.Cache == nil {
		fmt.Println("Warning: Redis cache is not available, falling back to database")
	}

	return nil
}

func GetDb(d *Databse) *gorm.DB {
	return d.Db
}

func GetRandomProblem(db *Databse) (*modles.ProblemPropaty, error) {

	//Use chace if avalable
	if db.Cache != nil {
		return db.Cache.GetRandomproblemm()
	}

	//fall back to the db
	var problems []modles.ProblemPropaty
	if err := db.Db.Preload("TestCases").Preload("Examples").Find(&problems).Error; err != nil {
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
	db.Db.Model(&modles.ProblemPropaty{}).Count(&count)
	if count > 0 {
		fmt.Println("Problems already exist in database, skipping insertion")
		return nil
	}

	// Create problems array
	problems := []modles.ProblemPropaty{
		{
			Title:       "Two Sum",
			Description: "Given an array of integers nums and an integer target, return indices of the two numbers such that they add up to target.",
			Difficulty:  "easy",
			Examples: []modles.Example{
				{
					Input:          "nums = [2,7,11,15], target = 9",
					ExpectedOutput: "[0,1]",
				},
				{
					Input:          "nums = [3,2,4], target = 6",
					ExpectedOutput: "[1,2]",
				},
			},
			HaderFile: `#include <bits/stdc++.h> 
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
			TestCases: []modles.TestCaesPropaty{
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
			Difficulty:  "medium",
			Examples: []modles.Example{
				{
					Input:          "x = 123",
					ExpectedOutput: "321",
				},
				{
					Input:          "x = -123",
					ExpectedOutput: "-321",
				},
			},
			HaderFile: `#include <bits/stdc++.h>
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
			TestCases: []modles.TestCaesPropaty{
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

	// Insert all problems in a transaction
	tx := db.Db.Begin()
	if tx.Error != nil {
		return fmt.Errorf("failed to start transaction: %v", tx.Error)
	}

	for _, problem := range problems {
		fmt.Printf("Inserting problem: %s with difficulty: %s\n", problem.Title, problem.Difficulty)
		if err := tx.Create(&problem).Error; err != nil {
			tx.Rollback()
			return fmt.Errorf("failed to insert problem '%s': %v", problem.Title, err)
		}
		fmt.Printf("Problem '%s' inserted successfully with difficulty: %s\n", problem.Title, problem.Difficulty)
	}

	if err := tx.Commit().Error; err != nil {
		return fmt.Errorf("failed to commit transaction: %v", err)
	}

	// Clear cache after inserting new problems
	if db.Cache != nil {
		if err := db.Cache.ClearChace(); err != nil {
			fmt.Printf("Warning: failed to clear cache: %v\n", err)
		} else {
			fmt.Println("Cache cleared after inserting new problems")
		}
	}

	return nil
}
