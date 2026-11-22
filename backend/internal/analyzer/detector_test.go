package analyzer

import (
	"os"
	"path/filepath"
	"testing"
)

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}
}

func TestDetectGoProject(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "go.mod"), "module example.com/demo\n\nrequire (\n github.com/gin-gonic/gin v1.12.0\n gorm.io/gorm v1.31.0\n)\n")
	writeFile(t, filepath.Join(root, "main.go"), "package main\n")
	writeFile(t, filepath.Join(root, "internal", "service", "service.go"), "package service\n")
	writeFile(t, filepath.Join(root, "node_modules", "ignored.js"), "ignored")

	profile, err := NewDetector(100).Detect(root)
	if err != nil {
		t.Fatalf("detect: %v", err)
	}
	if profile.PrimaryLanguage != "Go" {
		t.Fatalf("expected primary language Go, got %s", profile.PrimaryLanguage)
	}
	if !containsString(profile.Frameworks, "Gin") || !containsString(profile.Frameworks, "GORM") {
		t.Fatalf("expected Gin and GORM, got %#v", profile.Frameworks)
	}
	if profile.BuildCommand != "go build ./..." || profile.TestCommand != "go test ./..." {
		t.Fatalf("unexpected commands: build=%q test=%q", profile.BuildCommand, profile.TestCommand)
	}
	if profile.FileCount != 3 {
		t.Fatalf("expected 3 files, got %d", profile.FileCount)
	}
}

func TestDetectReactProject(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "package.json"), `{
  "scripts": {"build": "vite build", "test": "vitest"},
  "dependencies": {"react": "19", "vite": "7"}
}`)
	writeFile(t, filepath.Join(root, "pnpm-lock.yaml"), "lockfileVersion: 9\n")
	writeFile(t, filepath.Join(root, "src", "App.tsx"), "export function App() { return null }\n")

	profile, err := NewDetector(100).Detect(root)
	if err != nil {
		t.Fatalf("detect: %v", err)
	}
	if profile.PackageManager != "pnpm" {
		t.Fatalf("expected pnpm, got %s", profile.PackageManager)
	}
	if !containsString(profile.Frameworks, "React") || !containsString(profile.Frameworks, "Vite") {
		t.Fatalf("expected React and Vite, got %#v", profile.Frameworks)
	}
	if profile.BuildCommand != "pnpm run build" || profile.TestCommand != "pnpm run test" {
		t.Fatalf("unexpected commands: build=%q test=%q", profile.BuildCommand, profile.TestCommand)
	}
}
