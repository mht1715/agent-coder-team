package domain

import (
	"time"

	"github.com/google/uuid"
)

const (
	SenderTypeUser        = "user"
	SenderTypeMainAgent   = "main_agent"
	SenderTypeWorkerAgent = "worker_agent"
	SenderTypeSystem      = "system"
)

type Message struct {
	ID             uuid.UUID      `json:"id"`
	ConversationID uuid.UUID      `json:"conversation_id"`
	SenderType     string         `json:"sender_type"`
	SenderID       *uuid.UUID     `json:"sender_id,omitempty"`
	Content        string         `json:"content"`
	ContentFormat  string         `json:"content_format"`
	Summary        string         `json:"summary,omitempty"`
	Metadata       map[string]any `json:"metadata,omitempty"`
	CreatedAt      time.Time      `json:"created_at"`
}
