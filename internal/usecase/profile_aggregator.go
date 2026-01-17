package usecase

import (
	"context"
	"log"
	"sync"
	"time"

	"github.com/google/uuid"

	"profile-aggregator/internal/domain"
)

type ProfileUseCase interface {
	GetProfile(ctx context.Context, clientID string, id uuid.UUID) (*domain.Profile, error)
	SetClientSources(clientID string, sourceNames []string)
	SetClientFields(clientID string, fieldNames []string)
}

type profileAggregator struct {
	sources       []domain.DataSource
	sourceTimeout time.Duration
	cache         domain.ProfileCache
	clientSources map[string][]string
	clientFields  map[string][]string
}

func NewProfileAggregator(timeout time.Duration, cache domain.ProfileCache, sources ...domain.DataSource) ProfileUseCase {
	return &profileAggregator{
		sources:       sources,
		sourceTimeout: timeout,
		cache:         cache,
		clientSources: make(map[string][]string),
		clientFields:  make(map[string][]string),
	}
}

func (uc *profileAggregator) SetClientSources(clientID string, sourceNames []string) {
	uc.clientSources[clientID] = sourceNames
}

func (uc *profileAggregator) SetClientFields(clientID string, fieldNames []string) {
	uc.clientFields[clientID] = fieldNames
}

func (uc *profileAggregator) GetProfile(ctx context.Context, clientID string, id uuid.UUID) (*domain.Profile, error) {
	if uc.cache != nil {
		cacheCtx, cancelCache := context.WithTimeout(ctx, 50*time.Millisecond)
		cached, err := uc.cache.Get(cacheCtx, clientID, id)
		cancelCache()
		if err == nil && cached != nil {
			return cached, nil
		}
		if err != nil && err != context.DeadlineExceeded {
			log.Printf("cache get error: %v", err)
		}
	}

	best := make(map[string]domain.DataPoint)
	var mu sync.Mutex
	var wg sync.WaitGroup

	allowedSources := uc.getSourcesForClient(clientID)

	for _, src := range allowedSources {
		wg.Add(1)
		go func(s domain.DataSource) {
			defer wg.Done()
			sctx, cancel := context.WithTimeout(ctx, uc.sourceTimeout)
			defer cancel()

			data, err := s.Fetch(sctx, id)
			if err != nil {
				log.Printf("source %s error: %v", s.Name(), err)
				return
			}

			mu.Lock()
			for k, dp := range data {
				if cur, ok := best[k]; !ok || dp.Priority < cur.Priority {
					best[k] = dp
				}
			}
			mu.Unlock()
		}(src)
	}

	wg.Wait()

	profile := domain.NewProfile(id)
	for k, dp := range best {
		switch k {
		case "name":
			if val, ok := dp.Value.(string); ok {
				profile.Name = val
			}
		case "email":
			if val, ok := dp.Value.(string); ok {
				profile.Email = val
			}
		case "avatar_url":
			if val, ok := dp.Value.(string); ok {
				profile.AvatarURL = val
			}
		default:
			if uc.isFieldAllowed(clientID, k) {
				profile.Fields[k] = dp.Value
			}
		}
	}

	if uc.cache != nil {
		_ = uc.cache.Set(ctx, clientID, profile, 0)
	}

	return profile, nil
}

func (uc *profileAggregator) getSourcesForClient(clientID string) []domain.DataSource {
	names, ok := uc.clientSources[clientID]
	if !ok {
		return uc.sources
	}

	var filtered []domain.DataSource
	for _, src := range uc.sources {
		for _, name := range names {
			if src.Name() == name {
				filtered = append(filtered, src)
				break
			}
		}
	}
	return filtered
}

func (uc *profileAggregator) isFieldAllowed(clientID string, fieldName string) bool {
	allowedFields, ok := uc.clientFields[clientID]
	if !ok {
		return true
	}

	for _, f := range allowedFields {
		if f == fieldName {
			return true
		}
	}
	return false
}
