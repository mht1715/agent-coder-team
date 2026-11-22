package persistence

import (
	"context"
	"time"

	"backend/internal/domain"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type MessageRepository struct {
	db *gorm.DB
}

func NewMessageRepository(db *gorm.DB) *MessageRepository {
	return &MessageRepository{db: db}
}

func (r *MessageRepository) Create(ctx context.Context, message *domain.Message) error {
	return r.db.WithContext(ctx).Create(ptr(messageToModel(message))).Error
}

func (r *MessageRepository) ListByConversation(
	ctx context.Context,
	conversationID uuid.UUID,
	before *time.Time,
	limit int,
) ([]domain.Message, error) {
	var models []messageModel
	query := r.db.WithContext(ctx).
		Where("conversation_id = ?", conversationID)
	if before != nil {
		query = query.Where("created_at < ?", *before)
	}
	if err := query.
		Order("created_at DESC, id DESC").
		Limit(limit).
		Find(&models).Error; err != nil {
		return nil, err
	}

	messages := make([]domain.Message, 0, len(models))
	for _, model := range models {
		messages = append(messages, *messageFromModel(model))
	}
	return messages, nil
}
