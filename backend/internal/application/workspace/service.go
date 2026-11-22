package workspace

import workspaceinfra "backend/internal/workspace"

type BrowseEntry = workspaceinfra.DirectoryEntry
type BrowseResult = workspaceinfra.BrowseResult

type Service struct {
	validator *workspaceinfra.Validator
}

func NewService(validator *workspaceinfra.Validator) *Service {
	return &Service{validator: validator}
}

func (s *Service) Browse(path string) (*BrowseResult, error) {
	return s.validator.Browse(path)
}
