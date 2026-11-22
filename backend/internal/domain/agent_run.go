package domain

import (
	"time"

	"github.com/google/uuid"
)

const (
	AgentRunKindRequirement = "requirement"
	AgentRunKindPlanning    = "planning"
	AgentRunKindExecution   = "execution"

	AgentRunStatusQueued    = "queued"
	AgentRunStatusRunning   = "running"
	AgentRunStatusSucceeded = "succeeded"
	AgentRunStatusFailed    = "failed"
	AgentRunStatusCancelled = "cancelled"
)

type RunLog struct {
	At      time.Time `json:"at"`
	Level   string    `json:"level"`
	Message string    `json:"message"`
}

type AgentRun struct {
	ID             uuid.UUID  `json:"id"`
	ProjectID      uuid.UUID  `json:"project_id"`
	TaskID         *uuid.UUID `json:"task_id,omitempty"`
	ConversationID *uuid.UUID `json:"conversation_id,omitempty"`
	Kind           string     `json:"kind"`
	Status         string     `json:"status"`
	Title          string     `json:"title"`
	InputSummary   string     `json:"input_summary,omitempty"`
	OutputSummary  string     `json:"output_summary,omitempty"`
	Logs           []RunLog   `json:"logs"`
	StartedAt      *time.Time `json:"started_at,omitempty"`
	FinishedAt     *time.Time `json:"finished_at,omitempty"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
}
