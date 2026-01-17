package main

import (
	"log"
	"net/http"
	"time"

	"profile-aggregator/internal/infra/cache"
	"profile-aggregator/internal/infra/sources"
	"profile-aggregator/internal/transport/eventbus"
	"profile-aggregator/internal/transport/httpapi"
	"profile-aggregator/internal/usecase"
)

func main() {
	cfg := cache.RedisConfig{
		Addr:     "localhost:6379",
		Password: "",
		DB:       0,
		UseTLS:   true,
	}
	pc := cache.NewRedisCache(cfg)

	uc := usecase.NewProfileAggregator(200*time.Millisecond, pc, sources.Source1{}, sources.Source2{}, sources.Source3{}, sources.Source4{})

	ebConsumer := eventbus.NewEventBusConsumer(uc)
	_ = ebConsumer

	http.HandleFunc("/profile", httpapi.ProfileHandler(uc))
	http.HandleFunc("/profile/", httpapi.ProfileHandler(uc))

	addr := ":8080"
	log.Printf("listening on %s", addr)
	if err := http.ListenAndServe(addr, nil); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
