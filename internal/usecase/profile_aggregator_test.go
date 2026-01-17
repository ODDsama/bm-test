package usecase

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"profile-aggregator/internal/domain"

	"github.com/google/uuid"
)

type slowSource struct {
	delay time.Duration
	name  string
}

func (s *slowSource) Name() string { return s.name }
func (s *slowSource) Fetch(ctx context.Context, id uuid.UUID) (map[string]domain.DataPoint, error) {
	select {
	case <-time.After(s.delay):
		return map[string]domain.DataPoint{
			"name": {Value: s.name, Priority: 1},
		}, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

func TestGetProfileParallelism(t *testing.T) {
	s1 := &slowSource{delay: 100 * time.Millisecond, name: "s1"}
	s2 := &slowSource{delay: 100 * time.Millisecond, name: "s2"}
	s3 := &slowSource{delay: 100 * time.Millisecond, name: "s3"}

	uc := NewProfileAggregator(200*time.Millisecond, nil, s1, s2, s3)

	start := time.Now()
	_, err := uc.GetProfile(context.Background(), "test-client", uuid.New())
	duration := time.Since(start)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if duration > 150*time.Millisecond {
		t.Errorf("expected parallel execution (~100ms), but took %v", duration)
	}
}

type multiFieldSource struct {
	name string
}

func (s multiFieldSource) Name() string { return s.name }
func (s multiFieldSource) Fetch(ctx context.Context, id uuid.UUID) (map[string]domain.DataPoint, error) {
	return map[string]domain.DataPoint{
		"name":       {Value: "Common Name", Priority: 0},
		"extra_one":  {Value: "Extra 1", Priority: 0},
		"extra_two":  {Value: "Extra 2", Priority: 0},
		"avatar_url": {Value: "http://avatar.com", Priority: 0},
	}, nil
}

func TestProfileFieldFiltering(t *testing.T) {
	src := multiFieldSource{name: "src1"}
	uc := NewProfileAggregator(200*time.Millisecond, nil, src)

	t.Run("Client with restricted fields", func(t *testing.T) {
		uc.SetClientFields("client1", []string{"extra_one"})
		p, err := uc.GetProfile(context.Background(), "client1", uuid.New())
		if err != nil {
			t.Fatal(err)
		}

		if p.Name != "Common Name" {
			t.Errorf("expected common field 'name' to be 'Common Name', got '%s'", p.Name)
		}
		if p.AvatarURL != "http://avatar.com" {
			t.Errorf("expected common field 'avatar_url' to be present")
		}
		if p.Fields["extra_one"] != "Extra 1" {
			t.Errorf("expected allowed extra field 'extra_one' to be present")
		}
		if _, ok := p.Fields["extra_two"]; ok {
			t.Errorf("did not expect restricted extra field 'extra_two' to be present")
		}
	})

	t.Run("Client with no restrictions", func(t *testing.T) {
		p, err := uc.GetProfile(context.Background(), "client2", uuid.New())
		if err != nil {
			t.Fatal(err)
		}

		if p.Fields["extra_one"] != "Extra 1" || p.Fields["extra_two"] != "Extra 2" {
			t.Errorf("expected all fields for client without restrictions")
		}
	})
}

func TestProfileCaching(t *testing.T) {
	id := uuid.New()
	src := multiFieldSource{name: "src1"}

	// Mock cache
	mc := &mockCache{storage: make(map[string][]byte)}

	uc := NewProfileAggregator(200*time.Millisecond, mc, src)

	// First call - should aggregate and cache
	p1, err := uc.GetProfile(context.Background(), "client1", id)
	if err != nil {
		t.Fatal(err)
	}

	if p1.Name != "Common Name" {
		t.Errorf("expected Name 'Common Name', got '%s'", p1.Name)
	}

	// Second call - should come from cache
	p2, err := uc.GetProfile(context.Background(), "client1", id)
	if err != nil {
		t.Fatal(err)
	}

	if p2.Name != p1.Name || p2.Email != p1.Email || p2.ID != p1.ID {
		t.Errorf("profile from cache does not match original: %+v vs %+v", p2, p1)
	}

	if p2.Fields["extra_one"] != "Extra 1" {
		t.Errorf("expected field 'extra_one' from cache, got %v", p2.Fields["extra_one"])
	}

	// Verify JSON structure for API (flat)
	resp := map[string]any{
		"id": p2.ID.String(),
	}
	if p2.Name != "" {
		resp["name"] = p2.Name
	}
	if p2.Email != "" {
		resp["email"] = p2.Email
	}
	if p2.AvatarURL != "" {
		resp["avatar_url"] = p2.AvatarURL
	}
	for k, v := range p2.Fields {
		resp[k] = v
	}

	jsonBytes, _ := json.Marshal(resp)
	var finalMap map[string]any
	json.Unmarshal(jsonBytes, &finalMap)

	if finalMap["name"] != "Common Name" {
		t.Errorf("expected flat 'name' to be 'Common Name', got %v", finalMap["name"])
	}
	if finalMap["extra_one"] != "Extra 1" {
		t.Errorf("expected flat 'extra_one' to be 'Extra 1', got %v", finalMap["extra_one"])
	}
	if finalMap["avatar_url"] != "http://avatar.com" {
		t.Errorf("expected flat 'avatar_url' to be 'http://avatar.com', got %v", finalMap["avatar_url"])
	}
	if _, ok := finalMap["fields"]; ok {
		t.Errorf("response should not contain 'fields' key, it must be flat")
	}
}

type mockCache struct {
	storage map[string][]byte
}

func (m *mockCache) Get(ctx context.Context, clientID string, id uuid.UUID) (*domain.Profile, error) {
	key := clientID + id.String()
	data, ok := m.storage[key]
	if !ok {
		return nil, nil
	}
	p := domain.NewProfile(id)
	if err := json.Unmarshal(data, p); err != nil {
		return nil, err
	}
	return p, nil
}

func (m *mockCache) Set(ctx context.Context, clientID string, profile *domain.Profile, ttl time.Duration) error {
	key := clientID + profile.ID.String()
	data, err := json.Marshal(profile)
	if err != nil {
		return err
	}
	m.storage[key] = data
	return nil
}

func (m *mockCache) DeleteOlderThan(ctx context.Context, olderThan time.Duration) error {
	return nil
}
