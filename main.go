package main

import (
	"context"
	"encoding/json"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/redis/go-redis/v9"
)

type Config struct {
	RedisAddr string
	RedisPassword string
	RedisDB   int
	ListName  string
}

type Notification struct {
	Command string                 `json:"command"`
	Args    []string               `json:"args,omitempty"`
	Env     map[string]string      `json:"env,omitempty"`
	Meta    map[string]interface{} `json:"meta,omitempty"`
}

func loadConfig() Config {
	redisAddr := os.Getenv("REDIS_ADDR")
	if redisAddr == "" {
		redisAddr = "localhost:6379"
	}

	listName := os.Getenv("REDIS_LIST_NAME")
	if listName == "" {
		listName = "poppit:notifications"
	}

	return Config{
		RedisAddr:     redisAddr,
		RedisPassword: os.Getenv("REDIS_PASSWORD"),
		RedisDB:       0,
		ListName:      listName,
	}
}

func main() {
	log.Println("Starting Poppit service...")

	config := loadConfig()
	log.Printf("Config: Redis=%s, List=%s", config.RedisAddr, config.ListName)

	// Create Redis client
	rdb := redis.NewClient(&redis.Options{
		Addr:     config.RedisAddr,
		Password: config.RedisPassword,
		DB:       config.RedisDB,
	})

	ctx := context.Background()

	// Test Redis connection
	if err := rdb.Ping(ctx).Err(); err != nil {
		log.Fatalf("Failed to connect to Redis: %v", err)
	}
	log.Println("Connected to Redis successfully")

	// Setup graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	// Start popping messages
	go func() {
		for {
			// BRPOP blocks until a message is available or timeout occurs
			result, err := rdb.BRPop(ctx, 5*time.Second, config.ListName).Result()
			if err == redis.Nil {
				// Timeout - no message available
				continue
			} else if err != nil {
				log.Printf("Error popping message: %v", err)
				time.Sleep(1 * time.Second)
				continue
			}

			// result[0] is the list name, result[1] is the message
			if len(result) < 2 {
				log.Println("Received empty result from Redis")
				continue
			}

			message := result[1]
			log.Println("=== New Notification ===")
			log.Printf("Raw message: %s", message)

			// Try to parse as JSON
			var notification Notification
			if err := json.Unmarshal([]byte(message), &notification); err != nil {
				log.Printf("Failed to parse JSON: %v", err)
				log.Println("========================")
				continue
			}

			// Pretty print the notification
			prettyJSON, err := json.MarshalIndent(notification, "", "  ")
			if err == nil {
				log.Printf("Parsed notification:\n%s", string(prettyJSON))
			}
			log.Println("========================")
		}
	}()

	// Wait for shutdown signal
	<-sigChan
	log.Println("\nShutting down gracefully...")
	
	// Close Redis connection
	if err := rdb.Close(); err != nil {
		log.Printf("Error closing Redis connection: %v", err)
	}
	
	log.Println("Shutdown complete")
}
