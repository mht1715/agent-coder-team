package git

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func runGitCommand(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %v: %v: %s", args, err, output)
	}
}

func TestChangeTrackerDetectsNewChanges(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git is not available")
	}
	dir := t.TempDir()
	runGitCommand(t, dir, "init")
	runGitCommand(t, dir, "config", "user.email", "test@example.com")
	runGitCommand(t, dir, "config", "user.name", "Test")
	path := filepath.Join(dir, "README.md")
	if err := os.WriteFile(path, []byte("one\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	runGitCommand(t, dir, "add", ".")
	runGitCommand(t, dir, "commit", "-m", "initial")

	tracker := NewChangeTracker()
	ctx := context.Background()
	before, err := tracker.Capture(ctx, dir)
	if err != nil {
		t.Fatalf("capture before: %v", err)
	}
	if err := os.WriteFile(path, []byte("two\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "new.txt"), []byte("new\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	after, err := tracker.Capture(ctx, dir)
	if err != nil {
		t.Fatalf("capture after: %v", err)
	}
	changed := tracker.ChangedBetween(before, after)
	foundModified := false
	foundNew := false
	for _, path := range changed {
		if path == "README.md" {
			foundModified = true
		}
		if path == "new.txt" {
			foundNew = true
		}
	}
	if !foundModified || !foundNew {
		t.Fatalf("expected modified and new files, got %v", changed)
	}
}

func TestChangeTrackerIgnoresNonRepository(t *testing.T) {
	tracker := NewChangeTracker()
	snapshot, err := tracker.Capture(context.Background(), t.TempDir())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(snapshot.Files) != 0 {
		t.Fatalf("expected empty snapshot, got %v", snapshot.Files)
	}
}
