package agent

import (
	"context"
	"errors"
	"testing"
	"time"

	"backend/internal/domain"

	"github.com/google/uuid"
)

type stubProfileRepository struct {
	profile *domain.ProjectProfile
	err     error
}

func (s *stubProfileRepository) Upsert(context.Context, *domain.ProjectProfile) error { return nil }
func (s *stubProfileRepository) FindByProjectID(context.Context, uuid.UUID) (*domain.ProjectProfile, error) {
	return s.profile, s.err
}

type stubSandbox struct{}

func (s *stubSandbox) Name() string     { return "stub" }
func (s *stubSandbox) Images() []string { return []string{"alpine:3.20", "golang:1.27-alpine"} }
func (s *stubSandbox) DefaultImage(language string) string {
	if language == "Go" {
		return "golang:1.27-alpine"
	}
	return "alpine:3.20"
}
func (s *stubSandbox) ValidateImage(image string) error {
	if image == "alpine:3.20" || image == "golang:1.27-alpine" {
		return nil
	}
	return errors.New("image is not allowed")
}
func (s *stubSandbox) Timeout() time.Duration { return time.Minute }
func (s *stubSandbox) Execute(context.Context, domain.SandboxRequest, func(domain.SandboxOutput)) (domain.SandboxResult, error) {
	return domain.SandboxResult{}, nil
}

func TestResolveExecutionSpecUsesProjectDefaults(t *testing.T) {
	service := &Service{
		profiles: &stubProfileRepository{profile: &domain.ProjectProfile{
			PrimaryLanguage: "Go",
			TestCommand:     "go test ./...",
		}},
		sandbox: &stubSandbox{},
	}
	command, image, err := service.resolveExecutionSpec(context.Background(), uuid.New(), "", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if command != "go test ./..." || image != "golang:1.27-alpine" {
		t.Fatalf("unexpected spec: command=%q image=%q", command, image)
	}
}

func TestResolveExecutionSpecHonorsExplicitValues(t *testing.T) {
	service := &Service{
		profiles: &stubProfileRepository{profile: &domain.ProjectProfile{PrimaryLanguage: "Go", TestCommand: "go test ./..."}},
		sandbox:  &stubSandbox{},
	}
	command, image, err := service.resolveExecutionSpec(context.Background(), uuid.New(), "echo custom", "alpine:3.20")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if command != "echo custom" || image != "alpine:3.20" {
		t.Fatalf("unexpected spec: command=%q image=%q", command, image)
	}
}

func TestResolveExecutionSpecRejectsUnknownImage(t *testing.T) {
	service := &Service{sandbox: &stubSandbox{}}
	_, _, err := service.resolveExecutionSpec(context.Background(), uuid.New(), "", "unknown:latest")
	if !errors.Is(err, ErrValidation) {
		t.Fatalf("expected validation error, got %v", err)
	}
}

func TestResolveExecutionSpecRequiresSandbox(t *testing.T) {
	service := &Service{}
	_, _, err := service.resolveExecutionSpec(context.Background(), uuid.New(), "", "")
	if !errors.Is(err, ErrSandboxUnavailable) {
		t.Fatalf("expected unavailable error, got %v", err)
	}
}
