package domain

import (
	"context"
	"time"

	"github.com/google/uuid"
)

type DataPoint struct {
	Value    any
	Priority int
}

type DataSource interface {
	Fetch(ctx context.Context, id uuid.UUID) (map[string]DataPoint, error)
	Name() string
}

type ProfileCache interface {
	Get(ctx context.Context, clientID string, id uuid.UUID) (*Profile, error)
	Set(ctx context.Context, clientID string, profile *Profile, ttl time.Duration) error
	DeleteOlderThan(ctx context.Context, olderThan time.Duration) error
}
