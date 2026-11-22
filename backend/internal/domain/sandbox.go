package domain

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// SandboxOutput is one streamed line from a sandboxed process.
type SandboxOutput struct {
	Stream string
	Text   string
}

// SandboxRequest describes one isolated execution. WorkspacePath is the path
// visible to the core service; the executor is responsible for mapping it into
// the sandbox without exposing the host filesystem.
type SandboxRequest struct {
	RunID         uuid.UUID
	WorkspacePath string
	Image         string
	Command       string
	Timeout       time.Duration
}

type SandboxResult struct {
	ExitCode  int
	Output    string
	Truncated bool
}

// SandboxExecutor runs untrusted project commands in an isolated environment.
type SandboxExecutor interface {
	Name() string
	Images() []string
	DefaultImage(language string) string
	ValidateImage(image string) error
	Timeout() time.Duration
	Execute(ctx context.Context, req SandboxRequest, emit func(SandboxOutput)) (SandboxResult, error)
}
