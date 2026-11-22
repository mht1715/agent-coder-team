package workspace

import (
	"os"
	"path/filepath"
	"testing"

	"backend/internal/config"
)

func TestValidatePath(t *testing.T) {
	root := t.TempDir()
	inside := filepath.Join(root, "demo")
	if err := os.MkdirAll(inside, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	outside := t.TempDir()

	validator, err := NewValidator(config.WorkspaceConfig{
		AllowedRoots: []string{root},
		MaxScanFiles: 100,
	})
	if err != nil {
		t.Fatalf("new validator: %v", err)
	}

	got, err := validator.ValidatePath(inside, true)
	if err != nil {
		t.Fatalf("expected inside path to be valid: %v", err)
	}
	if got != inside {
		t.Fatalf("expected %s, got %s", inside, got)
	}

	if _, err := validator.ValidatePath(outside, true); err == nil {
		t.Fatal("expected outside path to be rejected")
	}

	relative := "relative/path"
	if _, err := validator.ValidatePath(relative, false); err == nil {
		t.Fatal("expected relative path to be rejected")
	}

	newProject := filepath.Join(root, "new-project")
	if _, err := validator.ValidatePath(newProject, false); err != nil {
		t.Fatalf("expected new project path to be valid: %v", err)
	}
}
