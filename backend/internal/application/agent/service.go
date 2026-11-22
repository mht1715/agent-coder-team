package agent

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"backend/internal/domain"
	"backend/internal/event"

	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
)

var (
	ErrValidation         = errors.New("validation failed")
	ErrPlanExists         = errors.New("task plan already exists")
	ErrSandboxUnavailable = errors.New("sandbox execution is unavailable")
)

const (
	maxAgentMessageLength = 200_000
	agentChunkSize        = 24
	agentChunkDelay       = 18 * time.Millisecond
)

type Service struct {
	projects      domain.ProjectRepository
	profiles      domain.ProjectProfileRepository
	conversations domain.ConversationRepository
	messages      domain.MessageRepository
	requirements  domain.RequirementRepository
	tasks         domain.TaskRepository
	approvals     domain.ApprovalRepository
	runs          domain.AgentRunRepository
	sandbox       domain.SandboxExecutor
	changes       domain.WorkspaceChangeTracker
	broker        *event.Broker
	executionSem  chan struct{}

	conversationLocks sync.Map
}

func NewService(
	projects domain.ProjectRepository,
	profiles domain.ProjectProfileRepository,
	conversations domain.ConversationRepository,
	messages domain.MessageRepository,
	requirements domain.RequirementRepository,
	tasks domain.TaskRepository,
	approvals domain.ApprovalRepository,
	runs domain.AgentRunRepository,
	sandbox domain.SandboxExecutor,
	changes domain.WorkspaceChangeTracker,
	executionConcurrency int,
	broker *event.Broker,
) *Service {
	if executionConcurrency <= 0 {
		executionConcurrency = 1
	}
	return &Service{
		projects:      projects,
		profiles:      profiles,
		conversations: conversations,
		messages:      messages,
		requirements:  requirements,
		tasks:         tasks,
		approvals:     approvals,
		runs:          runs,
		sandbox:       sandbox,
		changes:       changes,
		broker:        broker,
		executionSem:  make(chan struct{}, executionConcurrency),
	}
}

// QueueUserMessageReply starts the main agent response without holding the HTTP request open.
func (s *Service) QueueUserMessageReply(ownerID, conversationID, messageID uuid.UUID) {
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
		defer cancel()
		if err := s.replyToUserMessage(ctx, ownerID, conversationID, messageID); err != nil {
			log.Error().Err(err).Str("conversation_id", conversationID.String()).Msg("main agent reply failed")
			s.publishConversation(conversationID, "agent.thinking", map[string]any{"active": false})
			s.publishConversation(conversationID, "stream.error", map[string]string{"message": "main agent reply failed"})
		}
	}()
}

func (s *Service) replyToUserMessage(ctx context.Context, ownerID, conversationID, messageID uuid.UUID) error {
	unlock := s.lockConversation(conversationID)
	defer unlock()

	conversation, err := s.conversations.FindByIDAndOwner(ctx, conversationID, ownerID)
	if err != nil {
		return err
	}
	project, err := s.projects.FindByIDAndOwner(ctx, conversation.ProjectID, ownerID)
	if err != nil {
		return err
	}

	messages, err := s.recentMessages(ctx, conversationID)
	if err != nil {
		return err
	}
	latestUserMessage := ""
	for i := len(messages) - 1; i >= 0; i-- {
		if messages[i].ID == messageID && messages[i].SenderType == domain.SenderTypeUser {
			latestUserMessage = messages[i].Content
			break
		}
	}
	if strings.TrimSpace(latestUserMessage) == "" {
		return nil
	}

	wantRequirement, wantPlan := detectIntent(latestUserMessage)

	var requirement *domain.RequirementDocument
	var plannedTasks []domain.Task
	var planApproval *domain.Approval
	if wantPlan {
		requirement, err = s.requirements.FindByProjectID(ctx, project.ID)
		if err != nil && !errors.Is(err, domain.ErrNotFound) {
			return err
		}
		if errors.Is(err, domain.ErrNotFound) {
			requirement, err = s.generateRequirementInternal(ctx, ownerID, project, conversationID)
			if err != nil {
				return err
			}
		}
	} else if wantRequirement {
		requirement, err = s.generateRequirementInternal(ctx, ownerID, project, conversationID)
		if err != nil {
			return err
		}
	}
	if wantPlan {
		replace := containsAny([]string{latestUserMessage}, "重新生成", "覆盖", "regenerate", "replace")
		plannedTasks, planApproval, err = s.generatePlanInternal(ctx, ownerID, project, replace, conversationID)
		if err != nil {
			log.Warn().Err(err).Str("project_id", project.ID.String()).Msg("automatic task plan generation skipped")
		}
	}

	content := composeReply(project, latestUserMessage, requirement, plannedTasks, planApproval, wantRequirement, wantPlan)
	if utf8.RuneCountInString(content) > maxAgentMessageLength {
		content = string([]rune(content)[:maxAgentMessageLength])
	}
	message, err := s.persistAgentMessage(ctx, conversationID, content)
	if err != nil {
		return err
	}

	s.publishConversation(conversationID, "agent.thinking", map[string]any{"active": true})
	deltaData := map[string]any{
		"message_id":      message.ID,
		"conversation_id": conversationID,
		"sender_type":     domain.SenderTypeMainAgent,
		"created_at":      message.CreatedAt,
	}
	for _, chunk := range chunkRunes(content, agentChunkSize) {
		payload := make(map[string]any, len(deltaData)+1)
		for key, value := range deltaData {
			payload[key] = value
		}
		payload["delta"] = chunk
		s.publishConversation(conversationID, "message.delta", payload)
		select {
		case <-ctx.Done():
			s.publishConversation(conversationID, "agent.thinking", map[string]any{"active": false})
			return ctx.Err()
		case <-time.After(agentChunkDelay):
		}
	}
	s.publishConversation(conversationID, "message.created", message)
	s.publishConversation(conversationID, "agent.thinking", map[string]any{"active": false})

	if requirement != nil {
		s.publishProject(project.ID, "requirement.updated", requirement)
	}
	if len(plannedTasks) > 0 {
		s.publishProject(project.ID, "tasks.updated", map[string]any{
			"project_id": project.ID,
			"count":      len(plannedTasks),
		})
	}
	if planApproval != nil {
		s.publishProject(project.ID, "approval.updated", planApproval)
	}
	return nil
}

func (s *Service) GenerateRequirement(
	ctx context.Context,
	ownerID, projectID uuid.UUID,
) (*domain.RequirementDocument, error) {
	project, err := s.projects.FindByIDAndOwner(ctx, projectID, ownerID)
	if err != nil {
		return nil, err
	}
	requirement, err := s.generateRequirementInternal(ctx, ownerID, project, uuid.Nil)
	if err != nil {
		return nil, err
	}
	if err := s.notifyRequirementGenerated(ctx, project, requirement); err != nil {
		log.Warn().Err(err).Msg("failed to write requirement notification message")
	}
	return requirement, nil
}

func (s *Service) UpdateRequirement(
	ctx context.Context,
	ownerID, projectID uuid.UUID,
	title, contentMD string,
) (*domain.RequirementDocument, error) {
	if _, err := s.projects.FindByIDAndOwner(ctx, projectID, ownerID); err != nil {
		return nil, err
	}
	requirement, err := s.requirements.FindByProjectID(ctx, projectID)
	if err != nil {
		return nil, err
	}
	title = strings.TrimSpace(title)
	contentMD = strings.TrimSpace(contentMD)
	if title == "" || len(title) > 255 {
		return nil, fmt.Errorf("%w: requirement title is required and must be at most 255 chars", ErrValidation)
	}
	if contentMD == "" {
		return nil, fmt.Errorf("%w: requirement content is required", ErrValidation)
	}
	if len(contentMD) > maxAgentMessageLength {
		return nil, fmt.Errorf("%w: requirement content is too long", ErrValidation)
	}
	requirement.Title = title
	requirement.ContentMD = contentMD
	requirement.Status = domain.RequirementStatusDraft
	requirement.Version++
	requirement.GeneratedBy = "user"
	requirement.ApprovedAt = nil
	requirement.UpdatedAt = time.Now().UTC()
	if err := s.requirements.Upsert(ctx, requirement); err != nil {
		return nil, err
	}
	updated, err := s.requirements.FindByProjectID(ctx, projectID)
	if err != nil {
		return nil, err
	}
	s.publishProject(projectID, "requirement.updated", updated)
	return updated, nil
}

func (s *Service) ApproveRequirement(
	ctx context.Context,
	ownerID, projectID uuid.UUID,
) (*domain.RequirementDocument, error) {
	if _, err := s.projects.FindByIDAndOwner(ctx, projectID, ownerID); err != nil {
		return nil, err
	}
	requirement, err := s.requirements.FindByProjectID(ctx, projectID)
	if err != nil {
		return nil, err
	}
	now := time.Now().UTC()
	requirement.Status = domain.RequirementStatusApproved
	requirement.ApprovedAt = &now
	requirement.UpdatedAt = now
	if err := s.requirements.Upsert(ctx, requirement); err != nil {
		return nil, err
	}
	updated, err := s.requirements.FindByProjectID(ctx, projectID)
	if err != nil {
		return nil, err
	}
	s.publishProject(projectID, "requirement.updated", updated)
	return updated, nil
}

func (s *Service) generateRequirementInternal(
	ctx context.Context,
	ownerID uuid.UUID,
	project *domain.Project,
	conversationID uuid.UUID,
) (*domain.RequirementDocument, error) {
	messages, err := s.recentMessagesForProject(ctx, project.ID)
	if err != nil {
		return nil, err
	}
	var profile *domain.ProjectProfile
	if found, profileErr := s.profiles.FindByProjectID(ctx, project.ID); profileErr == nil {
		profile = found
	}

	existing, existingErr := s.requirements.FindByProjectID(ctx, project.ID)
	if existingErr != nil && !errors.Is(existingErr, domain.ErrNotFound) {
		return nil, existingErr
	}
	now := time.Now().UTC()
	requirementID, err := uuid.NewV7()
	if err != nil {
		return nil, err
	}
	version := 1
	createdAt := now
	if existing != nil {
		requirementID = existing.ID
		version = existing.Version + 1
		createdAt = existing.CreatedAt
	}
	title, content := buildRequirementDocument(project, profile, messages)

	run := s.newRun(project.ID, domain.AgentRunKindRequirement, "生成产品需求文档")
	if conversationID != uuid.Nil {
		run.ConversationID = &conversationID
	}
	run.InputSummary = fmt.Sprintf("基于 %d 条对话消息生成 PRD", len(messages))
	run.Logs = append(run.Logs, domain.RunLog{At: now, Level: "info", Message: "已读取项目信息、技术画像和对话上下文"})
	if err := s.runs.Create(ctx, run); err != nil {
		return nil, err
	}

	requirement := &domain.RequirementDocument{
		ID:          requirementID,
		ProjectID:   project.ID,
		Title:       title,
		ContentMD:   content,
		Status:      domain.RequirementStatusDraft,
		Version:     version,
		GeneratedBy: domain.SenderTypeMainAgent,
		CreatedAt:   createdAt,
		UpdatedAt:   now,
	}
	if err := s.requirements.Upsert(ctx, requirement); err != nil {
		s.failRun(ctx, run, "写入需求文档失败: "+err.Error())
		return nil, err
	}
	s.completeRun(ctx, run, fmt.Sprintf("已生成 %s 第 %d 版", title, version))
	reloaded, err := s.requirements.FindByProjectID(ctx, project.ID)
	if err != nil {
		return nil, err
	}
	s.publishProject(project.ID, "run.updated", run)
	return reloaded, nil
}

func (s *Service) notifyRequirementGenerated(
	ctx context.Context,
	project *domain.Project,
	requirement *domain.RequirementDocument,
) error {
	content := fmt.Sprintf("产品需求文档《%s》第 %d 版已生成。请在产品需求页审阅并批准，然后再生成任务计划。", requirement.Title, requirement.Version)
	if isEnglishProject(project) {
		content = fmt.Sprintf("Requirement document \"%s\" (version %d) is ready. Review and approve it before generating the task plan.", requirement.Title, requirement.Version)
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
	return nil
}

func (s *Service) recentMessages(ctx context.Context, conversationID uuid.UUID) ([]domain.Message, error) {
	messages, err := s.messages.ListByConversation(ctx, conversationID, nil, 100)
	if err != nil {
		return nil, err
	}
	for left, right := 0, len(messages)-1; left < right; left, right = left+1, right-1 {
		messages[left], messages[right] = messages[right], messages[left]
	}
	return messages, nil
}

func (s *Service) recentMessagesForProject(ctx context.Context, projectID uuid.UUID) ([]domain.Message, error) {
	conversation, err := s.conversations.FindOrCreateProjectGroup(ctx, projectID)
	if err != nil {
		return nil, err
	}
	return s.recentMessages(ctx, conversation.ID)
}

func (s *Service) persistAgentMessage(ctx context.Context, conversationID uuid.UUID, content string) (*domain.Message, error) {
	now := time.Now().UTC()
	id, err := uuid.NewV7()
	if err != nil {
		return nil, err
	}
	message := &domain.Message{
		ID:             id,
		ConversationID: conversationID,
		SenderType:     domain.SenderTypeMainAgent,
		Content:        content,
		ContentFormat:  "markdown",
		Metadata:       map[string]any{"agent": "main"},
		CreatedAt:      now,
	}
	if err := s.messages.Create(ctx, message); err != nil {
		return nil, err
	}
	if err := s.conversations.TouchLastMessage(ctx, conversationID, now); err != nil {
		return nil, err
	}
	return message, nil
}

func (s *Service) newRun(projectID uuid.UUID, kind, title string) *domain.AgentRun {
	now := time.Now().UTC()
	id, _ := uuid.NewV7()
	return &domain.AgentRun{
		ID:           id,
		ProjectID:    projectID,
		Kind:         kind,
		Status:       domain.AgentRunStatusRunning,
		Title:        title,
		ChangedFiles: []string{},
		Logs:         []domain.RunLog{},
		StartedAt:    &now,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
}

func (s *Service) completeRun(ctx context.Context, run *domain.AgentRun, output string) {
	now := time.Now().UTC()
	run.Status = domain.AgentRunStatusSucceeded
	run.OutputSummary = output
	run.FinishedAt = &now
	run.UpdatedAt = now
	run.Logs = append(run.Logs, domain.RunLog{At: now, Level: "info", Message: output})
	if err := s.runs.Update(ctx, run); err != nil {
		log.Warn().Err(err).Str("run_id", run.ID.String()).Msg("failed to complete agent run")
	}
}

func (s *Service) failRun(ctx context.Context, run *domain.AgentRun, output string) {
	now := time.Now().UTC()
	run.Status = domain.AgentRunStatusFailed
	run.OutputSummary = output
	run.FinishedAt = &now
	run.UpdatedAt = now
	run.Logs = append(run.Logs, domain.RunLog{At: now, Level: "error", Message: output})
	_ = s.runs.Update(ctx, run)
}

func (s *Service) lockConversation(conversationID uuid.UUID) func() {
	value, _ := s.conversationLocks.LoadOrStore(conversationID, &sync.Mutex{})
	mutex := value.(*sync.Mutex)
	mutex.Lock()
	return mutex.Unlock
}

func (s *Service) publishConversation(conversationID uuid.UUID, eventType string, data any) {
	s.broker.Publish(event.ConversationTopic(conversationID), event.Event{
		Type:       eventType,
		OccurredAt: time.Now().UTC(),
		Data:       data,
	})
}

func (s *Service) publishProject(projectID uuid.UUID, eventType string, data any) {
	conversation, err := s.conversations.FindOrCreateProjectGroup(context.Background(), projectID)
	if err != nil {
		return
	}
	s.publishConversation(conversation.ID, eventType, data)
}

func chunkRunes(value string, size int) []string {
	if size <= 0 || value == "" {
		return []string{value}
	}
	runes := []rune(value)
	chunks := make([]string, 0, (len(runes)+size-1)/size)
	for start := 0; start < len(runes); start += size {
		end := start + size
		if end > len(runes) {
			end = len(runes)
		}
		chunks = append(chunks, string(runes[start:end]))
	}
	return chunks
}

func (s *Service) GetRequirement(ctx context.Context, ownerID, projectID uuid.UUID) (*domain.RequirementDocument, error) {
	if _, err := s.projects.FindByIDAndOwner(ctx, projectID, ownerID); err != nil {
		return nil, err
	}
	return s.requirements.FindByProjectID(ctx, projectID)
}
