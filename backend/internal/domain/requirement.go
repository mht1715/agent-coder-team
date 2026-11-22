package domain

import (
	"time"

	"github.com/google/uuid"
)

const (
	RequirementStatusDraft    = "draft"
	RequirementStatusApproved = "approved"
)

type RequirementDocument struct {
	ID          uuid.UUID  `json:"id"`
	ProjectID   uuid.UUID  `json:"project_id"`
	Title       string     `json:"title"`
	ContentMD   string     `json:"content_md"`
	Status      string     `json:"status"`
	Version     int        `json:"version"`
	GeneratedBy string     `json:"generated_by"`
	ApprovedAt  *time.Time `json:"approved_at,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}
