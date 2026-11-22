package sandbox

import (
	"strings"
	"testing"
	"time"

	"backend/internal/config"
	"backend/internal/domain"

	"github.com/google/uuid"
)

func testExecutor(anchor string) *DockerExecutor {
	return &DockerExecutor{cfg: config.SandboxConfig{
		Driver:                 "docker",
		Network:                "none",
		CPUs:                   2,
		MemoryMB:               2048,
		PidsLimit:              256,
		TimeoutSeconds:         60,
		WorkspaceAnchor:        anchor,
		ContainerWorkspaceRoot: "/workspaces",
		DefaultImage:           "alpine:3.20",
		Images:                 []string{"alpine:3.20", "golang:1.27-alpine"},
	}}
}

func TestBuildDockerArgsIsolatesSandbox(t *testing.T) {
	executor := testExecutor("act-workspace-anchor")
	req := domain.SandboxRequest{
		RunID:   uuid.MustParse("0190d0b0-0000-7000-8000-000000000001"),
		Image:   "alpine:3.20",
		Command: "echo ok",
		Timeout: time.Minute,
	}
	args := executor.buildDockerArgs(req, "act-run-test", "/workspaces/demo", []string{"--volumes-from", "act-workspace-anchor"})
	joined := strings.Join(args, " ")
	for _, required := range []string{"--network none", "--cap-drop ALL", "--security-opt no-new-privileges:true", "--pids-limit 256", "--volumes-from act-workspace-anchor", "--workdir /workspaces/demo", "--entrypoint /bin/sh"} {
		if !strings.Contains(joined, required) {
			t.Fatalf("expected docker args to contain %q, got %s", required, joined)
		}
	}
	if strings.Contains(joined, "docker.sock") {
		t.Fatal("sandbox must not receive the Docker socket")
	}
}

func TestWorkspaceMappingWithAnchor(t *testing.T) {
	executor := testExecutor("act-workspace-anchor")
	workdir, mounts, err := executor.workspaceMapping("/workspaces/demo-go")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if workdir != "/workspaces/demo-go" {
		t.Fatalf("unexpected workdir: %s", workdir)
	}
	if strings.Join(mounts, " ") != "--volumes-from act-workspace-anchor" {
		t.Fatalf("unexpected mounts: %v", mounts)
	}
}

func TestWorkspaceMappingRejectsOutsideRoot(t *testing.T) {
	executor := testExecutor("act-workspace-anchor")
	if _, _, err := executor.workspaceMapping("/tmp/outside"); err == nil {
		t.Fatal("expected workspace outside root to be rejected")
	}
}

func TestValidateImageUsesAllowlist(t *testing.T) {
	executor := testExecutor("")
	if err := executor.ValidateImage("alpine:3.20"); err != nil {
		t.Fatalf("expected allowlisted image, got %v", err)
	}
	if err := executor.ValidateImage("evil/image:latest"); err == nil {
		t.Fatal("expected non-allowlisted image to be rejected")
	}
}

func TestDefaultImageByLanguage(t *testing.T) {
	executor := testExecutor("")
	if got := executor.DefaultImage("Go"); got != "golang:1.27-alpine" {
		t.Fatalf("unexpected Go image: %s", got)
	}
	if got := executor.DefaultImage("Unknown"); got != "alpine:3.20" {
		t.Fatalf("unexpected fallback image: %s", got)
	}
}
