package sources

import (
	"context"
	"time"

	"github.com/google/uuid"

	"profile-aggregator/internal/domain"
)

type Source1 struct{}

func (s Source1) Name() string { return "source1" }
func (s Source1) Fetch(ctx context.Context, id uuid.UUID) (map[string]domain.DataPoint, error) {
	select {
	case <-time.After(30 * time.Millisecond):
		return map[string]domain.DataPoint{
			"email": {Value: "test@test.com", Priority: 0},
			"name":  {Value: "Bar Dor", Priority: 2},
		}, nil
	case <-ctx.Done():
		return map[string]domain.DataPoint{}, ctx.Err()
	}
}

type Source2 struct{}

func (s Source2) Name() string { return "source2" }
func (s Source2) Fetch(ctx context.Context, id uuid.UUID) (map[string]domain.DataPoint, error) {
	select {
	case <-time.After(20 * time.Millisecond):
		return map[string]domain.DataPoint{
			"name": {Value: "John Foo", Priority: 0},
		}, nil
	case <-ctx.Done():
		return map[string]domain.DataPoint{}, ctx.Err()
	}
}

type Source3 struct{}

func (s Source3) Name() string { return "source3" }
func (s Source3) Fetch(ctx context.Context, id uuid.UUID) (map[string]domain.DataPoint, error) {
	select {
	case <-time.After(10 * time.Millisecond):
		return map[string]domain.DataPoint{
			"name":       {Value: "John Bar", Priority: 1},
			"avatar_url": {Value: "https://i.pravatar.cc/300", Priority: 0},
		}, nil
	case <-ctx.Done():
		return map[string]domain.DataPoint{}, ctx.Err()
	}
}

type Source4 struct{}

func (s Source4) Name() string { return "source4" }
func (s Source4) Fetch(ctx context.Context, id uuid.UUID) (map[string]domain.DataPoint, error) {
	select {
	case <-time.After(15 * time.Millisecond):
		return map[string]domain.DataPoint{
			"unknown": {Value: "alien", Priority: 0},
		}, nil
	case <-ctx.Done():
		return map[string]domain.DataPoint{}, ctx.Err()
	}
}
