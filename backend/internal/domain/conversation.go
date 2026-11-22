package domain

import (
	"time"

	"github.com/google/uuid"
)

const ConversationTypeProjectGroup = "project_group"

type Conversation struct {
	ID            uuid.UUID  `json:"id"`
	ProjectID     uuid.UUID  `json:"project_id"`
	Type          string     `json:"type"`
	Title         string     `json:"title"`
	LastMessageAt *time.Time `json:"last_message_at,omitempty"`
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`
}
