package domain

import "context"

// WorkspaceSnapshot captures the working-tree state relevant to an execution.
type WorkspaceSnapshot struct {
	Files    map[string]string
	DiffStat string
}

// WorkspaceChangeTracker compares workspace snapshots before and after a run.
type WorkspaceChangeTracker interface {
	Capture(ctx context.Context, workspacePath string) (WorkspaceSnapshot, error)
	ChangedBetween(before, after WorkspaceSnapshot) []string
}
