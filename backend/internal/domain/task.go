package domain

import (
	"time"

	"github.com/google/uuid"
)

const (
	TaskStatusBacklog    = "backlog"
	TaskStatusReady      = "ready"
	TaskStatusInProgress = "in_progress"
	TaskStatusReview     = "review"
	TaskStatusDone       = "done"
	TaskStatusBlocked    = "blocked"
	TaskStatusCancelled  = "cancelled"
)

type Task struct {
	ID                 uuid.UUID   `json:"id"`
	ProjectID          uuid.UUID   `json:"project_id"`
	TaskKey            string      `json:"task_key"`
	Title              string      `json:"title"`
	Description        string      `json:"description"`
	AcceptanceCriteria string      `json:"acceptance_criteria"`
	AgentRole          string      `json:"agent_role"`
	Priority           string      `json:"priority"`
	Status             string      `json:"status"`
	Position           int         `json:"position"`
	ExecutionCommand   string      `json:"execution_command,omitempty"`
	ExecutionImage     string      `json:"execution_image,omitempty"`
	Dependencies       []uuid.UUID `json:"dependencies"`
	StartedAt          *time.Time  `json:"started_at,omitempty"`
	CompletedAt        *time.Time  `json:"completed_at,omitempty"`
	CreatedAt          time.Time   `json:"created_at"`
	UpdatedAt          time.Time   `json:"updated_at"`
}

func IsValidTaskStatus(status string) bool {
	switch status {
	case TaskStatusBacklog, TaskStatusReady, TaskStatusInProgress, TaskStatusReview,
		TaskStatusDone, TaskStatusBlocked, TaskStatusCancelled:
		return true
	default:
		return false
	}
}

func CanTransitionTask(from, to string) bool {
	if from == to {
		return true
	}
	switch from {
	case TaskStatusBacklog:
		return to == TaskStatusReady || to == TaskStatusCancelled || to == TaskStatusBlocked
	case TaskStatusReady:
		return to == TaskStatusInProgress || to == TaskStatusBlocked || to == TaskStatusCancelled
	case TaskStatusInProgress:
		return to == TaskStatusReview || to == TaskStatusBlocked || to == TaskStatusCancelled
	case TaskStatusReview:
		return to == TaskStatusDone || to == TaskStatusInProgress || to == TaskStatusCancelled
	case TaskStatusBlocked:
		return to == TaskStatusReady || to == TaskStatusCancelled
	case TaskStatusDone, TaskStatusCancelled:
		return false
	default:
		return false
	}
}
