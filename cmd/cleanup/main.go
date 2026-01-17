package main

import (
	"context"
	"log"
	"time"

	"profile-aggregator/internal/infra/cache"
)

func main() {
	cfg := cache.RedisConfig{
		Addr:     "localhost:6379",
		Password: "",
		DB:       0,
		UseTLS:   true,
	}

	pc := cache.NewRedisCache(cfg)
	ctx := context.Background()

	log.Println("Starting cache cleanup...")

	err := pc.DeleteOlderThan(ctx, 4*time.Hour)
	if err != nil {
		log.Fatalf("Cleanup failed: %v", err)
	}

	log.Println("Cache cleanup completed successfully.")
}
