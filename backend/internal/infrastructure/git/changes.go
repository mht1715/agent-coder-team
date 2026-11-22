package git

import (
	"bytes"
	"context"
	"crypto/sha256"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"backend/internal/domain"
)

type ChangeTracker struct{}

func NewChangeTracker() *ChangeTracker { return &ChangeTracker{} }

func (t *ChangeTracker) Capture(ctx context.Context, workspacePath string) (domain.WorkspaceSnapshot, error) {
	snapshot := domain.WorkspaceSnapshot{Files: map[string]string{}}
	workspacePath = strings.TrimSpace(workspacePath)
	if workspacePath == "" {
		return snapshot, nil
	}
	inside, err := runGit(ctx, workspacePath, "rev-parse", "--is-inside-work-tree")
	if err != nil || strings.TrimSpace(inside) != "true" {
		return snapshot, nil
	}

	status, err := runGitBytes(ctx, workspacePath, "status", "--porcelain=v1", "-z")
	if err != nil {
		return snapshot, err
	}
	for _, path := range parseStatusPaths(status) {
		absolute := filepath.Join(workspacePath, filepath.FromSlash(path))
		hash, hashErr := fileHash(absolute)
		if hashErr != nil {
			hash = "missing"
		}
		snapshot.Files[path] = hash
	}
	if stat, statErr := runGit(ctx, workspacePath, "diff", "--stat"); statErr == nil {
		snapshot.DiffStat = strings.TrimSpace(stat)
	}
	return snapshot, nil
}

func (t *ChangeTracker) ChangedBetween(before, after domain.WorkspaceSnapshot) []string {
	changed := make([]string, 0)
	for path, afterHash := range after.Files {
		if before.Files[path] != afterHash {
			changed = append(changed, path)
		}
	}
	sort.Strings(changed)
	return changed
}

func runGit(ctx context.Context, dir string, args ...string) (string, error) {
	output, err := runGitBytes(ctx, dir, args...)
	return string(output), err
}

func runGitBytes(ctx context.Context, dir string, args ...string) ([]byte, error) {
	args = append([]string{"-C", dir}, args...)
	cmd := exec.CommandContext(ctx, "git", args...)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("git %s: %s", strings.Join(args[2:], " "), strings.TrimSpace(stderr.String()))
	}
	return output, nil
}

func parseStatusPaths(raw []byte) []string {
	parts := bytes.Split(raw, []byte{0})
	paths := make([]string, 0, len(parts))
	for i := 0; i < len(parts); i++ {
		entry := string(parts[i])
		if len(entry) < 4 {
			continue
		}
		status := entry[:2]
		path := entry[3:]
		if (strings.HasPrefix(status, "R") || strings.HasPrefix(status, "C")) && i+1 < len(parts) {
			i++
			path = string(parts[i])
		}
		if path != "" {
			paths = append(paths, path)
		}
	}
	return paths
}

func fileHash(path string) (string, error) {
	info, err := os.Stat(path)
	if err != nil {
		return "", err
	}
	if info.IsDir() {
		return "dir", nil
	}
	if info.Size() > 16*1024*1024 {
		return fmt.Sprintf("size:%d:mtime:%d", info.Size(), info.ModTime().UnixNano()), nil
	}
	content, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(content)
	return fmt.Sprintf("%x", sum[:]), nil
}
