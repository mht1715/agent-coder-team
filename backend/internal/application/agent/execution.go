package agent

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"backend/internal/domain"

	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
)

const (
	maxExecutionLogs = 600
	maxRunLogLine    = 4000
	logFlushInterval = 2 * time.Second
)

func (s *Service) resolveExecutionSpec(
	ctx context.Context,
	projectID uuid.UUID,
	command, image string,
) (string, string, error) {
	if s.sandbox == nil {
		return "", "", ErrSandboxUnavailable
	}
	var profile *domain.ProjectProfile
	if s.profiles != nil {
		if found, err := s.profiles.FindByProjectID(ctx, projectID); err == nil {
			profile = found
		}
	}
	command = strings.TrimSpace(command)
	if command == "" && profile != nil {
		command = strings.TrimSpace(profile.TestCommand)
	}
	if command == "" && profile != nil {
		command = strings.TrimSpace(profile.BuildCommand)
	}
	if command == "" {
		command = "pwd && find . -maxdepth 2 -type f -print"
	}
	if utf8.RuneCountInString(command) > 4000 {
		return "", "", fmt.Errorf("%w: execution command is too long", ErrValidation)
	}
	image = strings.TrimSpace(image)
	if image == "" {
		language := ""
		if profile != nil {
			language = profile.PrimaryLanguage
		}
		image = s.sandbox.DefaultImage(language)
		if image == "" {
			images := s.sandbox.Images()
			if len(images) > 0 {
				image = images[0]
			}
		}
	}
	if err := s.sandbox.ValidateImage(image); err != nil {
		return "", "", fmt.Errorf("%w: %s", ErrValidation, err.Error())
	}
	return command, image, nil
}

func (s *Service) queueExecution(ownerID, runID uuid.UUID) {
	timeout := 31 * time.Minute
	if s.sandbox != nil && s.sandbox.Timeout() > 0 {
		timeout = s.sandbox.Timeout() + 2*time.Minute
	}
	go func() {
		s.executionSem <- struct{}{}
		defer func() { <-s.executionSem }()
		defer func() {
			if recovered := recover(); recovered != nil {
				log.Error().Interface("panic", recovered).Str("run_id", runID.String()).Msg("sandbox execution panicked")
			}
		}()

		ctx, cancel := context.WithTimeout(context.Background(), timeout)
		defer cancel()
		if err := s.executeRun(ctx, ownerID, runID); err != nil {
			log.Error().Err(err).Str("run_id", runID.String()).Msg("sandbox execution failed")
		}
	}()
}

func (s *Service) executeRun(ctx context.Context, ownerID, runID uuid.UUID) error {
	run, err := s.runs.FindByIDAndOwner(ctx, runID, ownerID)
	if err != nil {
		return err
	}
	if run.Status != domain.AgentRunStatusQueued && run.Status != domain.AgentRunStatusRunning {
		return nil
	}
	project, err := s.projects.FindByIDAndOwner(ctx, run.ProjectID, ownerID)
	if err != nil {
		return s.failExecutionRun(run, ownerID, "读取项目信息失败："+err.Error())
	}
	if run.TaskID == nil {
		return s.failExecutionRun(run, ownerID, "执行记录没有关联任务")
	}
	task, err := s.tasks.FindByIDAndOwner(ctx, *run.TaskID, ownerID)
	if err != nil {
		return s.failExecutionRun(run, ownerID, "读取任务信息失败："+err.Error())
	}
	if s.sandbox == nil {
		return s.failExecutionRun(run, ownerID, "Docker 沙箱执行器未配置")
	}
	if strings.TrimSpace(task.ExecutionCommand) == "" || strings.TrimSpace(task.ExecutionImage) == "" {
		return s.failExecutionRun(run, ownerID, "任务没有可用的执行命令或执行镜像")
	}

	now := time.Now().UTC()
	run.Status = domain.AgentRunStatusRunning
	run.StartedAt = &now
	run.UpdatedAt = now
	run.Logs = append(run.Logs, domain.RunLog{At: now, Level: "info", Message: "Docker 沙箱已启动"})
	if err := s.runs.Update(ctx, run); err != nil {
		return err
	}
	s.publishProject(run.ProjectID, "run.updated", run)

	var before domain.WorkspaceSnapshot
	if s.changes != nil {
		snapshot, snapshotErr := s.changes.Capture(ctx, project.WorkingDir)
		if snapshotErr != nil {
			run.Logs = append(run.Logs, domain.RunLog{At: time.Now().UTC(), Level: "warn", Message: "执行前工作区快照失败：" + snapshotErr.Error()})
		} else {
			before = snapshot
		}
	}

	var logMu sync.Mutex
	lastFlush := time.Now()
	emit := func(output domain.SandboxOutput) {
		line := sanitizeRunLogLine(output.Text)
		if line == "" {
			return
		}
		logMu.Lock()
		defer logMu.Unlock()
		stream := strings.ToLower(strings.TrimSpace(output.Stream))
		if stream == "" {
			stream = "output"
		}
		run.Logs = append(run.Logs, domain.RunLog{
			At:      time.Now().UTC(),
			Level:   logLevelForStream(stream),
			Message: stream + " | " + line,
		})
		if len(run.Logs) > maxExecutionLogs {
			run.Logs = run.Logs[len(run.Logs)-maxExecutionLogs:]
		}
		if time.Since(lastFlush) >= logFlushInterval {
			lastFlush = time.Now().UTC()
			run.UpdatedAt = lastFlush
			if err := s.runs.Update(ctx, run); err == nil {
				s.publishProject(run.ProjectID, "run.updated", run)
			}
		}
	}

	result, executionErr := s.sandbox.Execute(ctx, domain.SandboxRequest{
		RunID:         run.ID,
		WorkspacePath: project.WorkingDir,
		Image:         task.ExecutionImage,
		Command:       task.ExecutionCommand,
		Timeout:       s.sandbox.Timeout(),
	}, emit)

	changedFiles := []string{}
	var after domain.WorkspaceSnapshot
	if s.changes != nil {
		snapshot, snapshotErr := s.changes.Capture(context.Background(), project.WorkingDir)
		if snapshotErr != nil {
			run.Logs = append(run.Logs, domain.RunLog{At: time.Now().UTC(), Level: "warn", Message: "执行后工作区快照失败：" + snapshotErr.Error()})
		} else {
			after = snapshot
			changedFiles = s.changes.ChangedBetween(before, after)
			if strings.TrimSpace(after.DiffStat) != "" {
				run.Logs = append(run.Logs, domain.RunLog{At: time.Now().UTC(), Level: "info", Message: "git diff --stat:\n" + after.DiffStat})
			}
		}
	}

	finishedAt := time.Now().UTC()
	exitCode := result.ExitCode
	run.ExitCode = &exitCode
	run.ChangedFiles = changedFiles
	run.FinishedAt = &finishedAt
	run.UpdatedAt = finishedAt
	if executionErr != nil {
		run.Status = domain.AgentRunStatusFailed
		run.OutputSummary = "沙箱执行失败：" + executionErr.Error()
		run.Logs = append(run.Logs, domain.RunLog{At: finishedAt, Level: "error", Message: run.OutputSummary})
	} else if result.ExitCode != 0 {
		run.Status = domain.AgentRunStatusFailed
		run.OutputSummary = fmt.Sprintf("沙箱执行失败，退出码 %d，变更文件 %d 个。", result.ExitCode, len(changedFiles))
		run.Logs = append(run.Logs, domain.RunLog{At: finishedAt, Level: "error", Message: run.OutputSummary})
	} else {
		run.Status = domain.AgentRunStatusSucceeded
		run.OutputSummary = fmt.Sprintf("沙箱执行成功，退出码 0，变更文件 %d 个。", len(changedFiles))
		run.Logs = append(run.Logs, domain.RunLog{At: finishedAt, Level: "info", Message: run.OutputSummary})
	}

	persistCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := s.runs.Update(persistCtx, run); err != nil {
		return err
	}
	if task.Status == domain.TaskStatusInProgress {
		if run.Status == domain.AgentRunStatusSucceeded {
			task.Status = domain.TaskStatusReview
		} else {
			task.Status = domain.TaskStatusBlocked
		}
		task.UpdatedAt = finishedAt
		if err := s.tasks.Update(persistCtx, task); err != nil {
			log.Warn().Err(err).Str("task_id", task.ID.String()).Msg("failed to update task after sandbox execution")
		}
		s.publishProject(task.ProjectID, "tasks.updated", map[string]any{"project_id": task.ProjectID, "task_id": task.ID})
	}
	s.publishProject(run.ProjectID, "run.updated", run)
	return nil
}

func (s *Service) failExecutionRun(run *domain.AgentRun, ownerID uuid.UUID, message string) error {
	now := time.Now().UTC()
	exitCode := -1
	run.Status = domain.AgentRunStatusFailed
	run.OutputSummary = message
	run.ExitCode = &exitCode
	run.FinishedAt = &now
	run.UpdatedAt = now
	run.Logs = append(run.Logs, domain.RunLog{At: now, Level: "error", Message: message})
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := s.runs.Update(ctx, run); err != nil {
		return err
	}
	if run.TaskID != nil {
		task, err := s.tasks.FindByIDAndOwner(ctx, *run.TaskID, ownerID)
		if err == nil && task.Status == domain.TaskStatusInProgress {
			task.Status = domain.TaskStatusBlocked
			task.UpdatedAt = now
			_ = s.tasks.Update(ctx, task)
			s.publishProject(task.ProjectID, "tasks.updated", map[string]any{"project_id": task.ProjectID, "task_id": task.ID})
		}
	}
	s.publishProject(run.ProjectID, "run.updated", run)
	return fmt.Errorf("%s", message)
}

func sanitizeRunLogLine(value string) string {
	value = strings.ReplaceAll(value, "\x00", "")
	value = strings.TrimSpace(value)
	if utf8.RuneCountInString(value) > maxRunLogLine {
		value = string([]rune(value)[:maxRunLogLine]) + "..."
	}
	return value
}

func logLevelForStream(stream string) string {
	if stream == "stderr" {
		return "warn"
	}
	return "info"
}
