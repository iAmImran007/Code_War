package database

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/iAmImran007/Code_War/pkg/modles"
)

type Redis struct {
	client *redis.Client
	ctx    context.Context
	db     *Databse
}

func NewServerChace(db *Databse) *Redis {
	rdb := redis.NewClient(&redis.Options{
		Addr:     "localhost:6379",
		Password: "",
		DB:       0,
	})

	ctx := context.Background()
	_, err := rdb.Ping(ctx).Result()
	if err != nil {
		log.Panicf("Redis connection feild: %v", err)
		return nil
	}

	fmt.Println("Conneted to redis successFully")

	return &Redis{
		client: rdb,
		ctx:    ctx,
		db:     db,
	}
}

// get all problem
func (r *Redis) GetAllProblems() ([]modles.ProblemPropaty, error) {
	// Try to get from cache first
	cachedData, err := r.client.Get(r.ctx, "all_problems_basic").Result()
	if err == nil {
		// Cache hit - unmarshal and return
		var problems []modles.ProblemPropaty
		if err := json.Unmarshal([]byte(cachedData), &problems); err == nil {
			fmt.Println("Cache HIT: Retrieved problems from Redis")
			return problems, nil
		}
	}

	// Cache miss - fetch from database
	fmt.Println("Cache MISS: Fetching problems from database")
	var problems []modles.ProblemPropaty
	if err := r.db.Db.Select("id", "title", "difficulty").Find(&problems).Error; err != nil {
		return nil, err
	}

	// Store in cache for 1 hour
	problemsJSON, err := json.Marshal(problems)
	if err == nil {
		r.client.Set(r.ctx, "all_problems_basic", problemsJSON, time.Hour)
		fmt.Println("Problems cached in Redis")
	}

	return problems, nil
}




//get random problems
func (r *Redis) GetRandomproblemm() (*modles.ProblemPropaty, error) {
	// Try to get full problems from cache first
	cachedData, err := r.client.Get(r.ctx, "all_problems_full").Result()
	if err == nil {
		// Cache hit - unmarshal and return
		var problems []modles.ProblemPropaty
		if err := json.Unmarshal([]byte(cachedData), &problems); err == nil && len(problems) > 0 {
			fmt.Println("Cache HIT: Retrieved full problems from Redis")
			rand.Seed(time.Now().UnixNano())
			randomIndex := rand.Intn(len(problems))
			return &problems[randomIndex], nil
		}
	}

	// Cache miss - fetch from database with all relations
	fmt.Println("Cache MISS: Fetching full problems from database")
	var problems []modles.ProblemPropaty
	if err := r.db.Db.Preload("TestCases").Preload("Examples").Find(&problems).Error; err != nil {
		return nil, fmt.Errorf("failed to fetch problems from database: %v", err)
	}

	if len(problems) == 0 {
		return nil, fmt.Errorf("no problems found in database")
	}

	// Store FULL problem data in cache for 1 hour
	problemsJSON, err := json.Marshal(problems)
	if err == nil {
		r.client.Set(r.ctx, "all_problems_full", problemsJSON, time.Hour)
		fmt.Println("Full problems cached in Redis")
	}

	// Get random problem
	rand.Seed(time.Now().UnixNano())
	randomIndex := rand.Intn(len(problems))
	return &problems[randomIndex], nil
}



//get random problems by id
func (r *Redis) GetProblemById(problemId uint) (*modles.ProblemPropaty, error) {
	cacheKey := fmt.Sprintf("problem_%d", problemId)

	// Try to get from cache first
	cacheData, err := r.client.Get(r.ctx, cacheKey).Result()
	if err == nil {
		var problem modles.ProblemPropaty
		if err := json.Unmarshal([]byte(cacheData), &problem); err == nil {
			fmt.Printf("Cache Hit: Retrieved problem %d from redis\n", problemId)
			return &problem, nil
		}
	}

	// Cache miss - fetch from database with full data
	fmt.Printf("Cache MISS: Fetching problem %d from database\n", problemId)
	var problem modles.ProblemPropaty
	if err := r.db.Db.Preload("TestCases").Preload("Examples").Where("id = ?", problemId).First(&problem).Error; err != nil {
		return nil, fmt.Errorf("problem with id %d not found: %v", problemId, err)
	}

	// Cache the full problem data
	problemJSON, err := json.Marshal(problem)
	if err == nil {
		r.client.Set(r.ctx, cacheKey, problemJSON, time.Hour)
		fmt.Printf("Problem %d cached in Redis\n", problemId)
	}

	return &problem, nil
}



//clear it 
func (r *Redis) ClearChace() error {
	return r.client.FlushDB(r.ctx).Err()
}

// clear specific problem chace
func (r *Redis) ClearproblemCache(prblemId uint) error {
	cacheKey := fmt.Sprintf("problem_%d", prblemId)
	r.client.Del(r.ctx, cacheKey)
	r.client.Del(r.ctx, "all_problems_besic")
	r.client.Del(r.ctx, "all_problems_full")
	return nil
}
