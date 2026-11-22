package agent

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"backend/internal/domain"

	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
)

func (s *Service) GeneratePlan(
	ctx context.Context,
	ownerID, projectID uuid.UUID,
	replace bool,
) ([]domain.Task, *domain.Approval, error) {
	project, err := s.projects.FindByIDAndOwner(ctx, projectID, ownerID)
	if err != nil {
		return nil, nil, err
	}
	tasks, approval, err := s.generatePlanInternal(ctx, ownerID, project, replace, uuid.Nil)
	if err != nil {
		return nil, nil, err
	}
	if err := s.notifyPlanGenerated(ctx, project, tasks, approval); err != nil {
		log.Warn().Err(err).Msg("failed to write plan notification message")
	}
	return tasks, approval, nil
}

func (s *Service) generatePlanInternal(
	ctx context.Context,
	ownerID uuid.UUID,
	project *domain.Project,
	replace bool,
	conversationID uuid.UUID,
) ([]domain.Task, *domain.Approval, error) {
	requirement, err := s.requirements.FindByProjectID(ctx, project.ID)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return nil, nil, fmt.Errorf("%w: generate and review the requirement document before creating a task plan", ErrValidation)
		}
		return nil, nil, err
	}
	existing, err := s.tasks.ListByProject(ctx, project.ID)
	if err != nil {
		return nil, nil, err
	}
	if len(existing) > 0 && !replace {
		return nil, nil, ErrPlanExists
	}

	var profile *domain.ProjectProfile
	if found, profileErr := s.profiles.FindByProjectID(ctx, project.ID); profileErr == nil {
		profile = found
	}

	run := s.newRun(project.ID, domain.AgentRunKindPlanning, "生成任务 DAG")
	if conversationID != uuid.Nil {
		run.ConversationID = &conversationID
	}
	run.InputSummary = "基于已生成的需求文档拆解任务"
	run.Logs = append(run.Logs, domain.RunLog{At: time.Now().UTC(), Level: "info", Message: "已读取需求文档和项目技术画像"})
	if err := s.runs.Create(ctx, run); err != nil {
		return nil, nil, err
	}

	raw := buildTaskPlan(project, profile, requirement)
	tasks, err := buildDomainTasks(project.ID, requirement, raw)
	if err != nil {
		s.failRun(ctx, run, "任务 DAG 校验失败: "+err.Error())
		return nil, nil, err
	}
	if err := s.tasks.ReplacePlan(ctx, project.ID, tasks); err != nil {
		s.failRun(ctx, run, "写入任务计划失败: "+err.Error())
		return nil, nil, err
	}

	now := time.Now().UTC()
	approvalID, err := uuid.NewV7()
	if err != nil {
		return nil, nil, err
	}
	approval := &domain.Approval{
		ID:              approvalID,
		ProjectID:       project.ID,
		Kind:            domain.ApprovalKindPlanReview,
		Status:          domain.ApprovalStatusPending,
		Title:           "批准任务计划",
		Description:     fmt.Sprintf("主 Agent 已生成 %d 个任务。批准后，无阻塞依赖的任务将进入可执行状态。", len(tasks)),
		RequestedByType: domain.SenderTypeMainAgent,
		CreatedAt:       now,
		UpdatedAt:       now,
	}
	if isEnglishProject(project) {
		approval.Title = "Approve task plan"
		approval.Description = fmt.Sprintf("The main agent generated %d tasks. After approval, tasks without blocking dependencies become ready.", len(tasks))
	}
	if err := s.approvals.Create(ctx, approval); err != nil {
		s.failRun(ctx, run, "创建计划审批失败: "+err.Error())
		return nil, nil, err
	}

	s.completeRun(ctx, run, fmt.Sprintf("已生成 %d 个任务并提交计划审批", len(tasks)))
	storedTasks, err := s.tasks.ListByProject(ctx, project.ID)
	if err != nil {
		return nil, nil, err
	}
	s.publishProject(project.ID, "tasks.updated", map[string]any{"project_id": project.ID, "count": len(storedTasks)})
	s.publishProject(project.ID, "approval.updated", approval)
	s.publishProject(project.ID, "run.updated", run)
	return storedTasks, approval, nil
}

func (s *Service) notifyPlanGenerated(
	ctx context.Context,
	project *domain.Project,
	tasks []domain.Task,
	approval *domain.Approval,
) error {
	content := fmt.Sprintf("任务计划已生成，共 %d 个任务。请在审批与执行页批准计划；批准后，依赖已满足的任务会进入可执行状态。", len(tasks))
	if isEnglishProject(project) {
		content = fmt.Sprintf("The task plan is ready with %d tasks. Approve the plan in the Approvals tab; tasks with satisfied dependencies will become ready.", len(tasks))
	}
	conversation, err := s.conversations.FindOrCreateProjectGroup(ctx, project.ID)
	if err != nil {
		return err
	}
	message, err := s.persistAgentMessage(ctx, conversation.ID, content)
	if err != nil {
		return err
	}
	s.publishConversation(conversation.ID, "message.created", message)
	if approval != nil {
		s.publishConversation(conversation.ID, "approval.updated", approval)
	}
	return nil
}

func (s *Service) ListTasks(ctx context.Context, ownerID, projectID uuid.UUID) ([]domain.Task, error) {
	if _, err := s.projects.FindByIDAndOwner(ctx, projectID, ownerID); err != nil {
		return nil, err
	}
	return s.tasks.ListByProject(ctx, projectID)
}

func (s *Service) TransitionTask(
	ctx context.Context,
	ownerID, taskID uuid.UUID,
	toStatus, note string,
) (*domain.Task, error) {
	task, err := s.tasks.FindByIDAndOwner(ctx, taskID, ownerID)
	if err != nil {
		return nil, err
	}
	toStatus = strings.TrimSpace(toStatus)
	if !domain.IsValidTaskStatus(toStatus) {
		return nil, fmt.Errorf("%w: invalid task status", ErrValidation)
	}
	if task.Status == toStatus {
		return task, nil
	}
	if task.Status == domain.TaskStatusReady && toStatus == domain.TaskStatusInProgress {
		return nil, fmt.Errorf("%w: start execution through the approval workflow", ErrValidation)
	}
	if !domain.CanTransitionTask(task.Status, toStatus) {
		return nil, fmt.Errorf("%w: cannot move task from %s to %s", ErrValidation, task.Status, toStatus)
	}
	if toStatus == domain.TaskStatusReady {
		allTasks, listErr := s.tasks.ListByProject(ctx, task.ProjectID)
		if listErr != nil {
			return nil, listErr
		}
		if !dependenciesDone(*task, allTasks) {
			return nil, fmt.Errorf("%w: task dependencies are not complete", ErrValidation)
		}
	}

	now := time.Now().UTC()
	task.Status = toStatus
	task.UpdatedAt = now
	if toStatus == domain.TaskStatusInProgress && task.StartedAt == nil {
		task.StartedAt = &now
	}
	if toStatus == domain.TaskStatusDone {
		task.CompletedAt = &now
	}
	if toStatus == domain.TaskStatusReady || toStatus == domain.TaskStatusBacklog {
		task.CompletedAt = nil
	}
	if err := s.tasks.Update(ctx, task); err != nil {
		return nil, err
	}
	_ = note
	if toStatus == domain.TaskStatusDone {
		if err := s.unlockDependentTasks(ctx, task.ProjectID); err != nil {
			log.Warn().Err(err).Str("project_id", task.ProjectID.String()).Msg("failed to unlock dependent tasks")
		}
	}
	s.publishProject(task.ProjectID, "tasks.updated", map[string]any{"project_id": task.ProjectID, "task_id": task.ID})
	return task, nil
}

func (s *Service) unlockDependentTasks(ctx context.Context, projectID uuid.UUID) error {
	tasks, err := s.tasks.ListByProject(ctx, projectID)
	if err != nil {
		return err
	}
	for i := range tasks {
		task := &tasks[i]
		if task.Status != domain.TaskStatusBacklog {
			continue
		}
		if !dependenciesDone(*task, tasks) {
			continue
		}
		now := time.Now().UTC()
		task.Status = domain.TaskStatusReady
		task.UpdatedAt = now
		if err := s.tasks.Update(ctx, task); err != nil {
			return err
		}
	}
	return nil
}

func dependenciesDone(task domain.Task, allTasks []domain.Task) bool {
	statuses := make(map[uuid.UUID]string, len(allTasks))
	for _, candidate := range allTasks {
		statuses[candidate.ID] = candidate.Status
	}
	for _, dependencyID := range task.Dependencies {
		if statuses[dependencyID] != domain.TaskStatusDone {
			return false
		}
	}
	return true
}

func (s *Service) RequestTaskExecution(
	ctx context.Context,
	ownerID, taskID uuid.UUID,
) (*domain.Approval, error) {
	task, err := s.tasks.FindByIDAndOwner(ctx, taskID, ownerID)
	if err != nil {
		return nil, err
	}
	if task.Status != domain.TaskStatusReady {
		return nil, fmt.Errorf("%w: only ready tasks can request execution", ErrValidation)
	}
	approvals, err := s.approvals.ListByProject(ctx, task.ProjectID, domain.ApprovalStatusPending)
	if err != nil {
		return nil, err
	}
	for _, approval := range approvals {
		if approval.Kind == domain.ApprovalKindTaskExecution && approval.TaskID != nil && *approval.TaskID == task.ID {
			return nil, fmt.Errorf("%w: task already has a pending execution approval", ErrValidation)
		}
	}
	now := time.Now().UTC()
	id, err := uuid.NewV7()
	if err != nil {
		return nil, err
	}
	approval := &domain.Approval{
		ID:              id,
		ProjectID:       task.ProjectID,
		TaskID:          &task.ID,
		Kind:            domain.ApprovalKindTaskExecution,
		Status:          domain.ApprovalStatusPending,
		Title:           "批准开始执行：" + task.Title,
		Description:     task.Description,
		RequestedByType: domain.SenderTypeMainAgent,
		CreatedAt:       now,
		UpdatedAt:       now,
	}
	if err := s.approvals.Create(ctx, approval); err != nil {
		return nil, err
	}
	s.publishProject(task.ProjectID, "approval.updated", approval)
	return approval, nil
}

func (s *Service) ListApprovals(
	ctx context.Context,
	ownerID, projectID uuid.UUID,
	status string,
) ([]domain.Approval, error) {
	if _, err := s.projects.FindByIDAndOwner(ctx, projectID, ownerID); err != nil {
		return nil, err
	}
	if status != "" && status != domain.ApprovalStatusPending && status != domain.ApprovalStatusApproved && status != domain.ApprovalStatusRejected {
		return nil, fmt.Errorf("%w: invalid approval status", ErrValidation)
	}
	return s.approvals.ListByProject(ctx, projectID, status)
}

func (s *Service) DecideApproval(
	ctx context.Context,
	ownerID, approvalID uuid.UUID,
	decision, note string,
) (*domain.Approval, error) {
	approval, err := s.approvals.FindByIDAndOwner(ctx, approvalID, ownerID)
	if err != nil {
		return nil, err
	}
	if approval.Status != domain.ApprovalStatusPending {
		return nil, fmt.Errorf("%w: approval has already been decided", ErrValidation)
	}
	decision = strings.ToLower(strings.TrimSpace(decision))
	if decision != "approve" && decision != "reject" {
		return nil, fmt.Errorf("%w: decision must be approve or reject", ErrValidation)
	}
	now := time.Now().UTC()
	approval.DecidedBy = &ownerID
	approval.DecisionNote = strings.TrimSpace(note)
	approval.DecidedAt = &now
	approval.UpdatedAt = now

	if decision == "reject" {
		approval.Status = domain.ApprovalStatusRejected
		if err := s.approvals.Update(ctx, approval); err != nil {
			return nil, err
		}
		s.publishProject(approval.ProjectID, "approval.updated", approval)
		return approval, nil
	}

	approval.Status = domain.ApprovalStatusApproved
	switch approval.Kind {
	case domain.ApprovalKindPlanReview:
		if err := s.activateTaskPlan(ctx, approval.ProjectID); err != nil {
			return nil, err
		}
	case domain.ApprovalKindTaskExecution:
		if approval.TaskID == nil {
			return nil, fmt.Errorf("%w: task execution approval has no task", ErrValidation)
		}
		task, taskErr := s.tasks.FindByIDAndOwner(ctx, *approval.TaskID, ownerID)
		if taskErr != nil {
			return nil, taskErr
		}
		if task.Status != domain.TaskStatusReady {
			return nil, fmt.Errorf("%w: task is no longer ready", ErrValidation)
		}
		run, runErr := s.startExecutionRun(ctx, task)
		if runErr != nil {
			return nil, runErr
		}
		approval.RunID = &run.ID
		task.Status = domain.TaskStatusInProgress
		task.StartedAt = &now
		task.UpdatedAt = now
		if err := s.tasks.Update(ctx, task); err != nil {
			return nil, err
		}
		s.publishProject(task.ProjectID, "tasks.updated", map[string]any{"project_id": task.ProjectID, "task_id": task.ID})
		s.publishProject(task.ProjectID, "run.updated", run)
	default:
		return nil, fmt.Errorf("%w: unsupported approval kind", ErrValidation)
	}

	if err := s.approvals.Update(ctx, approval); err != nil {
		return nil, err
	}
	s.publishProject(approval.ProjectID, "approval.updated", approval)
	return approval, nil
}

func (s *Service) activateTaskPlan(ctx context.Context, projectID uuid.UUID) error {
	tasks, err := s.tasks.ListByProject(ctx, projectID)
	if err != nil {
		return err
	}
	now := time.Now().UTC()
	for i := range tasks {
		task := &tasks[i]
		if task.Status != domain.TaskStatusBacklog {
			continue
		}
		if !dependenciesDone(*task, tasks) {
			continue
		}
		task.Status = domain.TaskStatusReady
		task.UpdatedAt = now
		if err := s.tasks.Update(ctx, task); err != nil {
			return err
		}
	}
	s.publishProject(projectID, "tasks.updated", map[string]any{"project_id": projectID})
	return nil
}

func (s *Service) startExecutionRun(ctx context.Context, task *domain.Task) (*domain.AgentRun, error) {
	now := time.Now().UTC()
	run := s.newRun(task.ProjectID, domain.AgentRunKindExecution, "执行任务："+task.Title)
	run.TaskID = &task.ID
	run.Status = domain.AgentRunStatusQueued
	run.InputSummary = task.Description
	run.Logs = []domain.RunLog{
		{At: now, Level: "info", Message: "执行审批已通过，任务已进入执行队列"},
		{At: now, Level: "info", Message: "当前版本仅记录执行审批与运行状态，Docker 沙箱执行器将在后续阶段接入"},
	}
	if err := s.runs.Create(ctx, run); err != nil {
		return nil, err
	}
	return run, nil
}

func (s *Service) ListRuns(ctx context.Context, ownerID, projectID uuid.UUID) ([]domain.AgentRun, error) {
	if _, err := s.projects.FindByIDAndOwner(ctx, projectID, ownerID); err != nil {
		return nil, err
	}
	return s.runs.ListByProject(ctx, projectID)
}

func (s *Service) FinishRun(
	ctx context.Context,
	ownerID, runID uuid.UUID,
	status, outputSummary string,
) (*domain.AgentRun, error) {
	run, err := s.runs.FindByIDAndOwner(ctx, runID, ownerID)
	if err != nil {
		return nil, err
	}
	if run.Kind != domain.AgentRunKindExecution {
		return nil, fmt.Errorf("%w: only execution runs can be completed manually", ErrValidation)
	}
	if run.Status != domain.AgentRunStatusQueued && run.Status != domain.AgentRunStatusRunning {
		return nil, fmt.Errorf("%w: run is already finished", ErrValidation)
	}
	status = strings.ToLower(strings.TrimSpace(status))
	if status != domain.AgentRunStatusSucceeded && status != domain.AgentRunStatusFailed && status != domain.AgentRunStatusCancelled {
		return nil, fmt.Errorf("%w: run status must be succeeded, failed or cancelled", ErrValidation)
	}
	now := time.Now().UTC()
	run.Status = status
	run.OutputSummary = strings.TrimSpace(outputSummary)
	run.FinishedAt = &now
	run.UpdatedAt = now
	run.Logs = append(run.Logs, domain.RunLog{At: now, Level: runLogLevel(status), Message: "执行记录已更新：" + status})
	if err := s.runs.Update(ctx, run); err != nil {
		return nil, err
	}

	if run.TaskID != nil {
		task, taskErr := s.tasks.FindByIDAndOwner(ctx, *run.TaskID, ownerID)
		if taskErr == nil {
			switch status {
			case domain.AgentRunStatusSucceeded:
				task.Status = domain.TaskStatusReview
			case domain.AgentRunStatusFailed:
				task.Status = domain.TaskStatusBlocked
			case domain.AgentRunStatusCancelled:
				task.Status = domain.TaskStatusReady
			}
			task.UpdatedAt = now
			if task.Status == domain.TaskStatusReady {
				task.StartedAt = nil
			}
			_ = s.tasks.Update(ctx, task)
			s.publishProject(task.ProjectID, "tasks.updated", map[string]any{"project_id": task.ProjectID, "task_id": task.ID})
		}
	}
	s.publishProject(run.ProjectID, "run.updated", run)
	return run, nil
}

func runLogLevel(status string) string {
	if status == domain.AgentRunStatusFailed {
		return "error"
	}
	if status == domain.AgentRunStatusCancelled {
		return "warn"
	}
	return "info"
}
