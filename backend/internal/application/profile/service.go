package profile

import (
	"context"
	"errors"
	"fmt"

	"backend/internal/analyzer"
	"backend/internal/domain"
	"backend/internal/workspace"

	"github.com/google/uuid"
)

var ErrValidation = errors.New("validation failed")

type Service struct {
	projects   domain.ProjectRepository
	profiles   domain.ProjectProfileRepository
	workspaces *workspace.Validator
	detector   *analyzer.Detector
}

func NewService(
	projects domain.ProjectRepository,
	profiles domain.ProjectProfileRepository,
	workspaces *workspace.Validator,
	detector *analyzer.Detector,
) *Service {
	return &Service{
		projects:   projects,
		profiles:   profiles,
		workspaces: workspaces,
		detector:   detector,
	}
}

func (s *Service) Get(ctx context.Context, ownerID, projectID uuid.UUID) (*domain.ProjectProfile, error) {
	if _, err := s.projects.FindByIDAndOwner(ctx, projectID, ownerID); err != nil {
		return nil, err
	}
	return s.profiles.FindByProjectID(ctx, projectID)
}

func (s *Service) Scan(ctx context.Context, ownerID, projectID uuid.UUID) (*domain.ProjectProfile, error) {
	project, err := s.projects.FindByIDAndOwner(ctx, projectID, ownerID)
	if err != nil {
		return nil, err
	}

	workingDir, err := s.workspaces.ValidatePath(project.WorkingDir, true)
	if err != nil {
		return nil, fmt.Errorf("%w: %s", ErrValidation, err.Error())
	}

	projectProfile, err := s.detector.Detect(workingDir)
	if err != nil {
		return nil, fmt.Errorf("%w: %s", ErrValidation, err.Error())
	}
	projectProfile.ProjectID = projectID
	if err := s.profiles.Upsert(ctx, projectProfile); err != nil {
		return nil, err
	}
	return projectProfile, nil
}
