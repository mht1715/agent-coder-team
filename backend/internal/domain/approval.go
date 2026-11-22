package domain

import (
	"time"

	"github.com/google/uuid"
)

const (
	ApprovalKindPlanReview    = "plan_review"
	ApprovalKindTaskExecution = "task_execution"

	ApprovalStatusPending  = "pending"
	ApprovalStatusApproved = "approved"
	ApprovalStatusRejected = "rejected"
)

type Approval struct {
	ID              uuid.UUID  `json:"id"`
	ProjectID       uuid.UUID  `json:"project_id"`
	TaskID          *uuid.UUID `json:"task_id,omitempty"`
	RunID           *uuid.UUID `json:"run_id,omitempty"`
	Kind            string     `json:"kind"`
	Status          string     `json:"status"`
	Title           string     `json:"title"`
	Description     string     `json:"description"`
	RequestedByType string     `json:"requested_by_type"`
	DecidedBy       *uuid.UUID `json:"decided_by,omitempty"`
	DecisionNote    string     `json:"decision_note,omitempty"`
	DecidedAt       *time.Time `json:"decided_at,omitempty"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
}
