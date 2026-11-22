package workspace

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"backend/internal/config"
)

const maxBrowseEntries = 500

type DirectoryEntry struct {
	Name string `json:"name"`
	Path string `json:"path"`
}

type BrowseResult struct {
	Path      string           `json:"path"`
	Parent    string           `json:"parent,omitempty"`
	Root      string           `json:"root"`
	Roots     []string         `json:"roots"`
	Entries   []DirectoryEntry `json:"entries"`
	Truncated bool             `json:"truncated"`
}

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

func (v *Validator) Roots() []string {
	roots := make([]string, len(v.roots))
	copy(roots, v.roots)
	return roots
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

// Browse lists child directories under an allowed workspace root. The root
// itself may be browsed, but project creation still requires a child folder.
func (v *Validator) Browse(path string) (*BrowseResult, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		path = v.roots[0]
	}
	if !filepath.IsAbs(path) {
		return nil, fmt.Errorf("directory path must be absolute")
	}

	abs, err := filepath.Abs(path)
	if err != nil {
		return nil, fmt.Errorf("resolve directory path: %w", err)
	}
	evaluated, err := filepath.EvalSymlinks(filepath.Clean(abs))
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("directory does not exist")
		}
		return nil, fmt.Errorf("resolve directory path: %w", err)
	}

	root, ok := v.matchingRoot(evaluated, true)
	if !ok {
		return nil, fmt.Errorf("directory must be inside an allowed workspace root")
	}
	info, err := os.Stat(evaluated)
	if err != nil || !info.IsDir() {
		return nil, fmt.Errorf("directory does not exist")
	}

	children, err := os.ReadDir(evaluated)
	if err != nil {
		return nil, fmt.Errorf("read directory: %w", err)
	}
	entries := make([]DirectoryEntry, 0, len(children))
	directoryCount := 0
	for _, child := range children {
		if !child.IsDir() || child.Type()&os.ModeSymlink != 0 {
			continue
		}
		directoryCount++
		if len(entries) >= maxBrowseEntries {
			continue
		}
		entries = append(entries, DirectoryEntry{
			Name: child.Name(),
			Path: filepath.Join(evaluated, child.Name()),
		})
	}
	sort.Slice(entries, func(i, j int) bool {
		return strings.ToLower(entries[i].Name) < strings.ToLower(entries[j].Name)
	})

	result := &BrowseResult{
		Path:      evaluated,
		Root:      root,
		Roots:     v.Roots(),
		Entries:   entries,
		Truncated: directoryCount > len(entries),
	}
	if evaluated != root {
		result.Parent = filepath.Dir(evaluated)
	}
	return result, nil
}

func (v *Validator) matchingRoot(path string, allowRoot bool) (string, bool) {
	for _, root := range v.roots {
		if path == root {
			return root, allowRoot
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			continue
		}
		if rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
			return root, true
		}
	}
	return "", false
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
