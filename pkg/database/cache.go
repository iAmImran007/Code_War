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
	cachedData, err := r.client.Get(r.ctx, "all_problems").Result()
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
	if err := r.db.Db.Preload("TestCases").Find(&problems).Error; err != nil {
		return nil, err
	}

	// Store in cache for 1 hour
	problemsJSON, err := json.Marshal(problems)
	if err == nil {
		r.client.Set(r.ctx, "all_problems", problemsJSON, time.Hour)
		fmt.Println("Problems cached in Redis")
	}

	return problems, nil
}

// get random problem from cache
func (r *Redis) GetRandomproblemm() (*modles.ProblemPropaty, error) {
	problem, err := r.GetAllProblems()
	if err != nil {
		return nil, err
	}

	if len(problem) == 0 {
		return nil, fmt.Errorf("no problem found in database")
	}

	//get the random problem
	rand.Seed(time.Now().UnixNano())
	randomIndex := rand.Intn(len(problem))
	return &problem[randomIndex], nil

}

// getByid
func (r *Redis) GetProblemById(problemId uint) (*modles.ProblemPropaty, error) {
	cacheKey := fmt.Sprintf("problem_%d", problemId)

	cacheData, err := r.client.Get(r.ctx, cacheKey).Result()
	if err == nil {
		var problem modles.ProblemPropaty
		if err := json.Unmarshal([]byte(cacheData), &problem); err != nil {
			fmt.Printf("Chace Hit: Retrive problem %d from redis \n", problemId)
			return &problem, nil
		}
	}

	//cache missied
	problems, err := r.GetAllProblems()
	if err != nil {
		return nil, err
	}

	//find the specific problem
	for _, prblem := range problems {
		if prblem.ID == problemId {
			problemJSON, err := json.Marshal(prblem)
			if err == nil {
				r.client.Set(r.ctx, cacheKey, problemJSON, time.Hour)
			}
			return &prblem, nil
		}
	}

	return nil, fmt.Errorf("prolem is in the problem id %d not found", problemId)
}

// clear
func (r *Redis) ClearChace() error {
	return r.client.FlushDB(r.ctx).Err()
}

// clear specific problem chace
func (r *Redis) ClearproblemCache(prblemId uint) error {
	cacheKey := fmt.Sprintf("problem_%d", prblemId)
	r.client.Del(r.ctx, cacheKey)
	r.client.Del(r.ctx, "all_problems")
	return nil
}
