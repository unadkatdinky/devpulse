package cache

import (
	"context"
	"fmt"
	"log"

	"github.com/redis/go-redis/v9"
	"github.com/unadkatdinky/devpulse/internal/config"
)

// Connect creates and verifies a Redis connection.
// Returns a *redis.Client — pass this to anything that needs Redis.
func Connect(cfg *config.Config) *redis.Client {
	client := redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%s", cfg.RedisHost, cfg.RedisPort),
		Password: cfg.RedisPassword,
		DB:       cfg.RedisDB,
	})

	// Ping verifies the connection is actually working.
	// context.Background() is a default context — think of it as
	// "no deadline, no cancellation, just do it"
	if err := client.Ping(context.Background()).Err(); err != nil {
		log.Fatalf("❌ Failed to connect to Redis: %v", err)
	}

	log.Println("✅ Connected to Redis")
	return client
}