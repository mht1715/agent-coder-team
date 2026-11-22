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

func TestBrowse(t *testing.T) {
	root := t.TempDir()
	for _, name := range []string{"alpha", "beta"} {
		if err := os.MkdirAll(filepath.Join(root, name), 0o755); err != nil {
			t.Fatalf("mkdir: %v", err)
		}
	}
	if err := os.WriteFile(filepath.Join(root, "note.txt"), []byte("not a directory"), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}
	outside := t.TempDir()

	validator, err := NewValidator(config.WorkspaceConfig{
		AllowedRoots: []string{root},
		MaxScanFiles: 100,
	})
	if err != nil {
		t.Fatalf("new validator: %v", err)
	}

	result, err := validator.Browse(root)
	if err != nil {
		t.Fatalf("browse root: %v", err)
	}
	if len(result.Entries) != 2 {
		t.Fatalf("expected 2 directory entries, got %d", len(result.Entries))
	}
	if result.Entries[0].Name != "alpha" || result.Entries[1].Name != "beta" {
		t.Fatalf("unexpected entries: %#v", result.Entries)
	}
	if result.Parent != "" {
		t.Fatalf("root should not have a parent, got %q", result.Parent)
	}

	child, err := validator.Browse(filepath.Join(root, "alpha"))
	if err != nil {
		t.Fatalf("browse child: %v", err)
	}
	if child.Parent != root {
		t.Fatalf("expected parent %q, got %q", root, child.Parent)
	}

	if _, err := validator.Browse(outside); err == nil {
		t.Fatal("expected outside directory to be rejected")
	}
}
