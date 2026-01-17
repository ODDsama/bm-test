package eventbus

import (
	"context"
	"encoding/json"
	"log"
	"profile-aggregator/internal/usecase"

	"github.com/google/uuid"
)

type ProfileWarmupEvent struct {
	ClientID  string `json:"client_id"`
	ProfileID string `json:"profile_id"`
}

type EventBusConsumer struct {
	uc usecase.ProfileUseCase
}

func NewEventBusConsumer(uc usecase.ProfileUseCase) *EventBusConsumer {
	return &EventBusConsumer{uc: uc}
}

func (c *EventBusConsumer) HandleMessage(msg []byte) error {
	var event ProfileWarmupEvent
	if err := json.Unmarshal(msg, &event); err != nil {
		return err
	}

	id, err := uuid.Parse(event.ProfileID)
	if err != nil {
		return err
	}

	log.Printf("EventBus: warming up profile %s for client %s", event.ProfileID, event.ClientID)

	_, err = c.uc.GetProfile(context.Background(), event.ClientID, id)
	return err
}
