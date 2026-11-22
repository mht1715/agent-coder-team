package domain

import (
	"time"

	"github.com/google/uuid"
)

const (
	ProjectModeNew      = "new"
	ProjectModeExisting = "existing"

	ProjectStatusDraft  = "draft"
	ProjectStatusActive = "active"
)

type Project struct {
	ID             uuid.UUID
	OwnerID        uuid.UUID
	Name           string
	Description    string
	WorkingDir     string
	ProjectMode    string
	Status         string
	OutputLanguage string
	CreatedAt      time.Time
	UpdatedAt      time.Time
}
