package sandbox

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os/exec"
	"path"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	"backend/internal/config"
	"backend/internal/domain"

	"github.com/google/uuid"
)

var imageNamePattern = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9._/:@-]{0,254}$`)

type DockerExecutor struct {
	cfg config.SandboxConfig

	checkOnce sync.Once
	checkErr  error
}

func NewDockerExecutor(cfg config.SandboxConfig) (*DockerExecutor, error) {
	if !strings.EqualFold(cfg.Driver, "docker") {
		return nil, fmt.Errorf("unsupported sandbox driver %q", cfg.Driver)
	}
	if strings.TrimSpace(cfg.DefaultImage) == "" {
		return nil, fmt.Errorf("sandbox default image is required")
	}
	if len(cfg.Images) == 0 {
		return nil, fmt.Errorf("sandbox image allowlist is empty")
	}
	if cfg.TimeoutSeconds <= 0 {
		cfg.TimeoutSeconds = 1800
	}
	if cfg.MaxOutputBytes <= 0 {
		cfg.MaxOutputBytes = 200_000
	}
	return &DockerExecutor{cfg: cfg}, nil
}

func (e *DockerExecutor) Name() string { return "docker" }

func (e *DockerExecutor) Timeout() time.Duration {
	return time.Duration(e.cfg.TimeoutSeconds) * time.Second
}

func (e *DockerExecutor) Images() []string {
	images := make([]string, len(e.cfg.Images))
	copy(images, e.cfg.Images)
	return images
}

func (e *DockerExecutor) DefaultImage(language string) string {
	switch strings.ToLower(strings.TrimSpace(language)) {
	case "go":
		return "golang:1.27-alpine"
	case "javascript", "typescript", "vue", "node", "node.js":
		return "node:22-alpine"
	case "python":
		return "python:3.13-alpine"
	case "java", "kotlin":
		return "eclipse-temurin:21-jdk-alpine"
	case "c#", "csharp", ".net", "dotnet":
		return "mcr.microsoft.com/dotnet/sdk:8.0-alpine"
	default:
		return e.cfg.DefaultImage
	}
}

func (e *DockerExecutor) ValidateImage(image string) error {
	image = strings.TrimSpace(image)
	if image == "" {
		return fmt.Errorf("sandbox image is required")
	}
	if !imageNamePattern.MatchString(image) {
		return fmt.Errorf("sandbox image name is invalid")
	}
	for _, allowed := range e.cfg.Images {
		if image == allowed {
			return nil
		}
	}
	return fmt.Errorf("sandbox image %q is not in the allowlist", image)
}

func (e *DockerExecutor) Execute(
	ctx context.Context,
	req domain.SandboxRequest,
	emit func(domain.SandboxOutput),
) (domain.SandboxResult, error) {
	if err := e.ValidateImage(req.Image); err != nil {
		return domain.SandboxResult{ExitCode: -1}, err
	}
	command := strings.TrimSpace(req.Command)
	if command == "" {
		return domain.SandboxResult{ExitCode: -1}, fmt.Errorf("sandbox command is required")
	}
	if req.Timeout <= 0 {
		req.Timeout = time.Duration(e.cfg.TimeoutSeconds) * time.Second
	}
	if err := e.checkDocker(ctx); err != nil {
		return domain.SandboxResult{ExitCode: -1}, err
	}

	workdir, mountArgs, err := e.workspaceMapping(req.WorkspacePath)
	if err != nil {
		return domain.SandboxResult{ExitCode: -1}, err
	}
	name := sandboxContainerName(req.RunID)
	args := e.buildDockerArgs(req, name, workdir, mountArgs)

	runCtx, cancel := context.WithTimeout(ctx, req.Timeout)
	defer cancel()

	cmd := exec.CommandContext(runCtx, "docker", args...)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return domain.SandboxResult{ExitCode: -1}, fmt.Errorf("open sandbox stdout: %w", err)
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return domain.SandboxResult{ExitCode: -1}, fmt.Errorf("open sandbox stderr: %w", err)
	}
	if err := cmd.Start(); err != nil {
		return domain.SandboxResult{ExitCode: -1}, fmt.Errorf("start sandbox: %w", err)
	}

	collector := newOutputCollector(e.cfg.MaxOutputBytes, emit)
	var readers sync.WaitGroup
	readers.Add(2)
	go func() {
		defer readers.Done()
		collector.read("stdout", stdout)
	}()
	go func() {
		defer readers.Done()
		collector.read("stderr", stderr)
	}()

	waitErr := cmd.Wait()
	readers.Wait()
	cleanupContainer(name)

	result := domain.SandboxResult{
		ExitCode:  0,
		Output:    collector.Output(),
		Truncated: collector.Truncated(),
	}
	if runCtx.Err() != nil {
		return result, fmt.Errorf("sandbox execution timed out after %s", req.Timeout)
	}
	if waitErr == nil {
		return result, nil
	}
	if exitErr, ok := waitErr.(*exec.ExitError); ok {
		result.ExitCode = exitErr.ExitCode()
		return result, nil
	}
	return result, fmt.Errorf("wait for sandbox: %w", waitErr)
}

func (e *DockerExecutor) checkDocker(ctx context.Context) error {
	e.checkOnce.Do(func() {
		if _, err := exec.LookPath("docker"); err != nil {
			e.checkErr = fmt.Errorf("docker CLI is not available: %w", err)
			return
		}
		checkCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
		defer cancel()
		output, err := exec.CommandContext(checkCtx, "docker", "version", "--format", "{{.Server.Version}}").CombinedOutput()
		if err != nil {
			e.checkErr = fmt.Errorf("docker daemon is unavailable: %s", strings.TrimSpace(string(output)))
		}
	})
	return e.checkErr
}

func (e *DockerExecutor) workspaceMapping(workspacePath string) (string, []string, error) {
	workspacePath = strings.TrimSpace(workspacePath)
	if workspacePath == "" {
		return "", nil, fmt.Errorf("workspace path is required")
	}
	anchor := strings.TrimSpace(e.cfg.WorkspaceAnchor)
	if anchor != "" {
		root := strings.TrimSpace(e.cfg.ContainerWorkspaceRoot)
		if root == "" {
			return "", nil, fmt.Errorf("sandbox container workspace root is required when workspace anchor is configured")
		}
		root = path.Clean(strings.ReplaceAll(root, "\\", "/"))
		containerPath := path.Clean(strings.ReplaceAll(workspacePath, "\\", "/"))
		if containerPath == root || !strings.HasPrefix(containerPath, root+"/") {
			return "", nil, fmt.Errorf("workspace %q is outside the sandbox workspace root", workspacePath)
		}
		return containerPath, []string{"--volumes-from", anchor}, nil
	}

	abs, err := filepath.Abs(workspacePath)
	if err != nil {
		return "", nil, fmt.Errorf("resolve workspace path: %w", err)
	}
	return "/workspace", []string{
		"--mount", "type=bind,source=" + abs + ",target=/workspace,rw",
	}, nil
}

func (e *DockerExecutor) buildDockerArgs(
	req domain.SandboxRequest,
	name, workdir string,
	mountArgs []string,
) []string {
	args := []string{
		"run", "--rm", "--init",
		"--name", name,
		"--network", valueOrDefault(e.cfg.Network, "none"),
		"--cpus", fmt.Sprintf("%.2f", valueOrDefaultFloat(e.cfg.CPUs, 2)),
		"--memory", fmt.Sprintf("%dm", valueOrDefaultInt(e.cfg.MemoryMB, 2048)),
		"--pids-limit", fmt.Sprintf("%d", valueOrDefaultInt(e.cfg.PidsLimit, 256)),
		"--cap-drop", "ALL",
		"--security-opt", "no-new-privileges:true",
		"--tmpfs", "/tmp:rw,nosuid,nodev,size=512m",
		"--workdir", workdir,
		"--env", "CI=1",
		"--env", "NO_COLOR=1",
		"--env", "TERM=dumb",
		"--env", "HOME=/tmp",
		"--label", "act.managed=true",
		"--label", "act.run_id=" + req.RunID.String(),
	}
	args = append(args, mountArgs...)
	args = append(args, "--entrypoint", "/bin/sh", req.Image, "-lc", strings.TrimSpace(req.Command))
	return args
}

type outputCollector struct {
	maxBytes int
	emit     func(domain.SandboxOutput)

	mu        sync.Mutex
	buffer    strings.Builder
	truncated bool
	lines     int
}

func newOutputCollector(maxBytes int, emit func(domain.SandboxOutput)) *outputCollector {
	return &outputCollector{maxBytes: maxBytes, emit: emit}
}

func (c *outputCollector) read(stream string, reader io.Reader) {
	scanner := bufio.NewScanner(reader)
	scanner.Buffer(make([]byte, 64*1024), 256*1024)
	for scanner.Scan() {
		line := scanner.Text()
		c.mu.Lock()
		if c.truncated {
			c.mu.Unlock()
			continue
		}
		if c.buffer.Len()+len(line)+1 > c.maxBytes {
			c.truncated = true
			c.mu.Unlock()
			continue
		}
		c.buffer.WriteString(line)
		c.buffer.WriteByte('\n')
		c.lines++
		emit := c.emit
		c.mu.Unlock()
		if emit != nil {
			emit(domain.SandboxOutput{Stream: stream, Text: line})
		}
	}
}

func (c *outputCollector) Output() string {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.truncated {
		return c.buffer.String() + "\n[output truncated by sandbox limit]\n"
	}
	return c.buffer.String()
}

func (c *outputCollector) Truncated() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.truncated
}

func sandboxContainerName(runID uuid.UUID) string {
	value := strings.ReplaceAll(runID.String(), "-", "")
	if len(value) > 24 {
		value = value[:24]
	}
	return "act-run-" + value
}

func cleanupContainer(name string) {
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()
	_ = exec.CommandContext(ctx, "docker", "rm", "-f", name).Run()
}

func valueOrDefault(value, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}

func valueOrDefaultInt(value, fallback int) int {
	if value <= 0 {
		return fallback
	}
	return value
}

func valueOrDefaultFloat(value, fallback float64) float64 {
	if value <= 0 {
		return fallback
	}
	return value
}
