package conversation

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"backend/internal/domain"
	"backend/internal/event"

	"github.com/google/uuid"
)

var ErrValidation = errors.New("validation failed")

const (
	defaultMessageLimit = 50
	maxMessageLimit     = 100
	maxMessageLength    = 200_000
)

type Service struct {
	projects      domain.ProjectRepository
	conversations domain.ConversationRepository
	messages      domain.MessageRepository
	broker        *event.Broker
}

func NewService(
	projects domain.ProjectRepository,
	conversations domain.ConversationRepository,
	messages domain.MessageRepository,
	broker *event.Broker,
) *Service {
	return &Service{
		projects:      projects,
		conversations: conversations,
		messages:      messages,
		broker:        broker,
	}
}

func (s *Service) ListProjectConversations(
	ctx context.Context,
	ownerID, projectID uuid.UUID,
) ([]domain.Conversation, error) {
	if _, err := s.projects.FindByIDAndOwner(ctx, projectID, ownerID); err != nil {
		return nil, err
	}
	if _, err := s.conversations.FindOrCreateProjectGroup(ctx, projectID); err != nil {
		return nil, err
	}
	return s.conversations.ListByProjectAndOwner(ctx, projectID, ownerID)
}

func (s *Service) GetConversation(ctx context.Context, ownerID, conversationID uuid.UUID) (*domain.Conversation, error) {
	return s.conversations.FindByIDAndOwner(ctx, conversationID, ownerID)
}

func (s *Service) ListMessages(
	ctx context.Context,
	ownerID, conversationID uuid.UUID,
	before *time.Time,
	limit int,
) ([]domain.Message, bool, error) {
	if _, err := s.conversations.FindByIDAndOwner(ctx, conversationID, ownerID); err != nil {
		return nil, false, err
	}
	if limit <= 0 {
		limit = defaultMessageLimit
	}
	if limit > maxMessageLimit {
		limit = maxMessageLimit
	}

	messages, err := s.messages.ListByConversation(ctx, conversationID, before, limit+1)
	if err != nil {
		return nil, false, err
	}
	hasMore := len(messages) > limit
	if hasMore {
		messages = messages[:limit]
	}
	return messages, hasMore, nil
}

func (s *Service) Subscribe(conversationID uuid.UUID) (<-chan event.Event, func()) {
	return s.broker.Subscribe(event.ConversationTopic(conversationID))
}

func (s *Service) SendMessage(
	ctx context.Context,
	ownerID, conversationID uuid.UUID,
	content, format string,
) (*domain.Message, error) {
	if _, err := s.conversations.FindByIDAndOwner(ctx, conversationID, ownerID); err != nil {
		return nil, err
	}

	content = strings.TrimSpace(content)
	if content == "" {
		return nil, fmt.Errorf("%w: message content is required", ErrValidation)
	}
	if len(content) > maxMessageLength {
		return nil, fmt.Errorf("%w: message is too long", ErrValidation)
	}
	if format == "" {
		format = "markdown"
	}
	if format != "markdown" && format != "text" {
		return nil, fmt.Errorf("%w: content format must be markdown or text", ErrValidation)
	}

	now := time.Now().UTC()
	messageID, err := uuid.NewV7()
	if err != nil {
		return nil, fmt.Errorf("generate message id: %w", err)
	}
	senderID := ownerID
	message := &domain.Message{
		ID:             messageID,
		ConversationID: conversationID,
		SenderType:     domain.SenderTypeUser,
		SenderID:       &senderID,
		Content:        content,
		ContentFormat:  format,
		Metadata:       map[string]any{},
		CreatedAt:      now,
	}
	if err := s.messages.Create(ctx, message); err != nil {
		return nil, err
	}
	if err := s.conversations.TouchLastMessage(ctx, conversationID, now); err != nil {
		return nil, err
	}

	s.broker.Publish(event.ConversationTopic(conversationID), event.Event{
		ID:         message.ID.String(),
		Type:       "message.created",
		OccurredAt: now,
		Data:       message,
	})
	return message, nil
}
