package domain

import (
	"github.com/google/uuid"
)

type Profile struct {
	ID        uuid.UUID      `json:"id"`
	Name      string         `json:"name,omitempty"`
	Email     string         `json:"email,omitempty"`
	AvatarURL string         `json:"avatar_url,omitempty"`
	Fields    map[string]any `json:"fields,omitempty"`
}

func NewProfile(id uuid.UUID) *Profile {
	return &Profile{
		ID:     id,
		Fields: make(map[string]any),
	}
}
