package config

import (
	"os"
	"path/filepath"
	"testing"
)

func validConfig() SysConfig {
	return SysConfig{
		App: AppConfig{
			Name:    "agent-coder-team",
			GinMode: "debug",
		},
		Server: ServerConfig{
			Host: "127.0.0.1",
			Port: 18080,
		},
		Database: DatabaseConfig{
			Host: "postgres",
			Port: 5432,
			User: "act",
			Name: "agent_coder_team",
		},
		Redis: RedisConfig{
			Host: "redis",
			Port: 6379,
		},
		Workspace: WorkspaceConfig{
			AllowedRoots: []string{"/workspaces"},
			MaxScanFiles: 20000,
		},
		Vector: VectorConfig{
			Dimension: 1536,
		},
		Scheduler: SchedulerConfig{
			Timezone: "Asia/Shanghai",
		},
	}
}

func TestValidate(t *testing.T) {
	t.Run("valid config", func(t *testing.T) {
		cfg := validConfig()
		if err := cfg.Validate(); err != nil {
			t.Fatalf("expected valid config, got %v", err)
		}
	})

	t.Run("invalid server port", func(t *testing.T) {
		cfg := validConfig()
		cfg.Server.Port = 0
		if err := cfg.Validate(); err == nil {
			t.Fatal("expected invalid server port error")
		}
	})

	t.Run("invalid timezone", func(t *testing.T) {
		cfg := validConfig()
		cfg.Scheduler.Timezone = "Invalid/Timezone"
		if err := cfg.Validate(); err == nil {
			t.Fatal("expected invalid timezone error")
		}
	})
}

func TestLoadSandboxFromEnvironment(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yml")
	if err := os.WriteFile(path, []byte("app:\n  name: test\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("CONFIG_PATH", path)
	t.Setenv("ACT_SANDBOX_WORKSPACE_ANCHOR", "act-workspace-anchor")
	t.Setenv("ACT_SANDBOX_CONTAINER_WORKSPACE_ROOT", "/workspaces")
	cfg, err := Load()
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	if cfg.Sandbox.WorkspaceAnchor != "act-workspace-anchor" {
		t.Fatalf("unexpected workspace anchor: %q", cfg.Sandbox.WorkspaceAnchor)
	}
	if cfg.Sandbox.ContainerWorkspaceRoot != "/workspaces" {
		t.Fatalf("unexpected container workspace root: %q", cfg.Sandbox.ContainerWorkspaceRoot)
	}
	if cfg.Sandbox.DefaultImage != "alpine:3.20" {
		t.Fatalf("unexpected default image: %q", cfg.Sandbox.DefaultImage)
	}
}
