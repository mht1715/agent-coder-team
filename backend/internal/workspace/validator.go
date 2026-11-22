package workspace

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"backend/internal/config"
)

type Validator struct {
	roots        []string
	maxScanFiles int
}

func NewValidator(cfg config.WorkspaceConfig) (*Validator, error) {
	roots := make([]string, 0, len(cfg.AllowedRoots))
	for _, root := range cfg.AllowedRoots {
		root = strings.TrimSpace(root)
		if root == "" {
			continue
		}
		abs, err := filepath.Abs(root)
		if err != nil {
			return nil, fmt.Errorf("resolve workspace root %q: %w", root, err)
		}
		if evaluated, err := filepath.EvalSymlinks(abs); err == nil {
			abs = evaluated
		}
		roots = append(roots, filepath.Clean(abs))
	}
	if len(roots) == 0 {
		return nil, fmt.Errorf("at least one workspace root is required")
	}
	maxScanFiles := cfg.MaxScanFiles
	if maxScanFiles <= 0 {
		maxScanFiles = 20000
	}
	return &Validator{roots: roots, maxScanFiles: maxScanFiles}, nil
}

func (v *Validator) MaxScanFiles() int {
	return v.maxScanFiles
}

func (v *Validator) ValidatePath(path string, requireExisting bool) (string, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return "", fmt.Errorf("working directory is required")
	}
	if !filepath.IsAbs(path) {
		return "", fmt.Errorf("working directory must be an absolute path")
	}

	abs, err := filepath.Abs(path)
	if err != nil {
		return "", fmt.Errorf("resolve working directory: %w", err)
	}
	abs = filepath.Clean(abs)

	evaluated := abs
	if resolved, err := filepath.EvalSymlinks(abs); err == nil {
		evaluated = filepath.Clean(resolved)
	} else if !os.IsNotExist(err) {
		return "", fmt.Errorf("resolve working directory: %w", err)
	} else if parent, parentErr := filepath.EvalSymlinks(filepath.Dir(abs)); parentErr == nil {
		evaluated = filepath.Join(parent, filepath.Base(abs))
	}

	if !v.insideAllowedRoot(evaluated) {
		return "", fmt.Errorf("working directory must be inside an allowed workspace root")
	}

	info, err := os.Stat(evaluated)
	if err != nil {
		if os.IsNotExist(err) && !requireExisting {
			parentInfo, parentErr := os.Stat(filepath.Dir(evaluated))
			if parentErr != nil || !parentInfo.IsDir() {
				return "", fmt.Errorf("parent directory does not exist")
			}
			return evaluated, nil
		}
		return "", fmt.Errorf("working directory does not exist")
	}
	if !info.IsDir() {
		return "", fmt.Errorf("working directory must be a directory")
	}
	return evaluated, nil
}

func (v *Validator) insideAllowedRoot(path string) bool {
	for _, root := range v.roots {
		if path == root {
			return false
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			continue
		}
		if rel == "." || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
			continue
		}
		return true
	}
	return false
}
