package cache

import (
	"bytes"
	"compress/gzip"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"

	"profile-aggregator/internal/domain"
)

type redisCache struct {
	client *redis.Client
}

type RedisConfig struct {
	Addr     string
	Password string
	DB       int
	UseTLS   bool
}

func NewRedisCache(cfg RedisConfig) domain.ProfileCache {
	opts := &redis.Options{
		Addr:     cfg.Addr,
		Password: cfg.Password,
		DB:       cfg.DB,
	}

	if cfg.UseTLS {
		opts.TLSConfig = &tls.Config{
			MinVersion: tls.VersionTLS12,
		}
	}

	rdb := redis.NewClient(opts)
	return &redisCache{client: rdb}
}

func (r *redisCache) Get(ctx context.Context, clientID string, id uuid.UUID) (*domain.Profile, error) {
	key := fmt.Sprintf("profile:%s:%s", clientID, id.String())
	val, err := r.client.Get(ctx, key).Bytes()
	if err == redis.Nil {
		return nil, nil
	} else if err != nil {
		return nil, err
	}

	decompressed, err := decompress(val)
	if err != nil {
		return nil, fmt.Errorf("failed to decompress: %w", err)
	}
	if decompressed == nil {
		return nil, nil
	}

	profile := domain.NewProfile(id)
	err = json.Unmarshal(decompressed, profile)
	if err != nil {
		return nil, err
	}

	return profile, nil
}

func (r *redisCache) Set(ctx context.Context, clientID string, profile *domain.Profile, ttl time.Duration) error {
	key := fmt.Sprintf("profile:%s:%s", clientID, profile.ID.String())
	val, err := json.Marshal(profile)
	if err != nil {
		return err
	}

	compressed, err := compress(val)
	if err != nil {
		return fmt.Errorf("failed to compress: %w", err)
	}

	err = r.client.Set(ctx, key, compressed, ttl).Err()
	if err != nil {
		return err
	}

	return r.client.ZAdd(ctx, "profiles_timestamps", redis.Z{
		Score:  float64(time.Now().Unix()),
		Member: key,
	}).Err()
}

func compress(data []byte) ([]byte, error) {
	var b bytes.Buffer
	w := gzip.NewWriter(&b)
	_, err := w.Write(data)
	if err != nil {
		return nil, err
	}
	if err := w.Close(); err != nil {
		return nil, err
	}
	return b.Bytes(), nil
}

func decompress(data []byte) ([]byte, error) {
	if len(data) == 0 {
		return nil, nil
	}
	r, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	defer r.Close()
	return io.ReadAll(r)
}

func (r *redisCache) DeleteOlderThan(ctx context.Context, olderThan time.Duration) error {
	cutoff := time.Now().Add(-olderThan).Unix()

	keys, err := r.client.ZRangeByScore(ctx, "profiles_timestamps", &redis.ZRangeBy{
		Min: "-inf",
		Max: fmt.Sprintf("%d", cutoff),
	}).Result()

	if err != nil {
		return err
	}

	if len(keys) == 0 {
		return nil
	}

	err = r.client.Del(ctx, keys...).Err()
	if err != nil {
		return err
	}

	return r.client.ZRem(ctx, "profiles_timestamps", keys).Err()
}
