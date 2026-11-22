package project

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"backend/internal/domain"

	"github.com/google/uuid"
)

var ErrValidation = errors.New("validation failed")

type CreateInput struct {
	Name           string
	Description    string
	WorkingDir     string
	ProjectMode    string
	OutputLanguage string
}

type UpdateInput struct {
	Name           *string
	Description    *string
	WorkingDir     *string
	ProjectMode    *string
	OutputLanguage *string
}

type Service struct {
	projects domain.ProjectRepository
}

func NewService(projects domain.ProjectRepository) *Service {
	return &Service{projects: projects}
}

func (s *Service) Create(ctx context.Context, ownerID uuid.UUID, input CreateInput) (*domain.Project, error) {
	name := strings.TrimSpace(input.Name)
	if name == "" || len(name) > 128 {
		return nil, fmt.Errorf("%w: project name is required and must be at most 128 chars", ErrValidation)
	}
	workingDir := strings.TrimSpace(input.WorkingDir)
	if workingDir == "" || len(workingDir) > 4096 {
		return nil, fmt.Errorf("%w: working directory is required", ErrValidation)
	}
	mode := input.ProjectMode
	if mode == "" {
		mode = domain.ProjectModeNew
	}
	if mode != domain.ProjectModeNew && mode != domain.ProjectModeExisting {
		return nil, fmt.Errorf("%w: project mode must be new or existing", ErrValidation)
	}
	language := input.OutputLanguage
	if language == "" {
		language = "zh-CN"
	}

	now := time.Now().UTC()
	id, err := uuid.NewV7()
	if err != nil {
		return nil, fmt.Errorf("generate project id: %w", err)
	}
	project := &domain.Project{
		ID:             id,
		OwnerID:        ownerID,
		Name:           name,
		Description:    strings.TrimSpace(input.Description),
		WorkingDir:     workingDir,
		ProjectMode:    mode,
		Status:         domain.ProjectStatusDraft,
		OutputLanguage: language,
		CreatedAt:      now,
		UpdatedAt:      now,
	}
	if err := s.projects.Create(ctx, project); err != nil {
		return nil, err
	}
	return project, nil
}

func (s *Service) List(ctx context.Context, ownerID uuid.UUID) ([]domain.Project, error) {
	return s.projects.ListByOwner(ctx, ownerID)
}

func (s *Service) Get(ctx context.Context, ownerID, projectID uuid.UUID) (*domain.Project, error) {
	return s.projects.FindByIDAndOwner(ctx, projectID, ownerID)
}

func (s *Service) Update(ctx context.Context, ownerID, projectID uuid.UUID, input UpdateInput) (*domain.Project, error) {
	project, err := s.projects.FindByIDAndOwner(ctx, projectID, ownerID)
	if err != nil {
		return nil, err
	}

	if input.Name != nil {
		name := strings.TrimSpace(*input.Name)
		if name == "" || len(name) > 128 {
			return nil, fmt.Errorf("%w: project name is required and must be at most 128 chars", ErrValidation)
		}
		project.Name = name
	}
	if input.Description != nil {
		project.Description = strings.TrimSpace(*input.Description)
	}
	if input.WorkingDir != nil {
		workingDir := strings.TrimSpace(*input.WorkingDir)
		if workingDir == "" || len(workingDir) > 4096 {
			return nil, fmt.Errorf("%w: working directory is required", ErrValidation)
		}
		project.WorkingDir = workingDir
	}
	if input.ProjectMode != nil {
		mode := *input.ProjectMode
		if mode != domain.ProjectModeNew && mode != domain.ProjectModeExisting {
			return nil, fmt.Errorf("%w: project mode must be new or existing", ErrValidation)
		}
		project.ProjectMode = mode
	}
	if input.OutputLanguage != nil && strings.TrimSpace(*input.OutputLanguage) != "" {
		project.OutputLanguage = strings.TrimSpace(*input.OutputLanguage)
	}
	project.UpdatedAt = time.Now().UTC()

	if err := s.projects.Update(ctx, project); err != nil {
		return nil, err
	}
	return project, nil
}
