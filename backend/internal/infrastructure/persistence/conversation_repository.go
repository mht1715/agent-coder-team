package persistence

import (
	"context"
	"errors"
	"time"

	"backend/internal/domain"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type ConversationRepository struct {
	db *gorm.DB
}

func NewConversationRepository(db *gorm.DB) *ConversationRepository {
	return &ConversationRepository{db: db}
}

func (r *ConversationRepository) FindOrCreateProjectGroup(ctx context.Context, projectID uuid.UUID) (*domain.Conversation, error) {
	conversation, err := r.findProjectGroup(ctx, projectID)
	if err == nil {
		return conversation, nil
	}
	if !errors.Is(err, domain.ErrNotFound) {
		return nil, err
	}

	now := time.Now().UTC()
	id, err := uuid.NewV7()
	if err != nil {
		return nil, err
	}
	model := conversationModel{
		ID:        id,
		ProjectID: projectID,
		Type:      domain.ConversationTypeProjectGroup,
		Title:     "Project Group",
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := r.db.WithContext(ctx).Create(&model).Error; err != nil {
		if errors.Is(err, gorm.ErrDuplicatedKey) {
			return r.findProjectGroup(ctx, projectID)
		}
		return nil, err
	}
	return conversationFromModel(model), nil
}

func (r *ConversationRepository) findProjectGroup(ctx context.Context, projectID uuid.UUID) (*domain.Conversation, error) {
	var model conversationModel
	if err := r.db.WithContext(ctx).
		Where("project_id = ? AND type = ?", projectID, domain.ConversationTypeProjectGroup).
		First(&model).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	return conversationFromModel(model), nil
}

func (r *ConversationRepository) FindByIDAndOwner(ctx context.Context, conversationID, ownerID uuid.UUID) (*domain.Conversation, error) {
	var model conversationModel
	if err := r.db.WithContext(ctx).
		Table("conversations").
		Select("conversations.*").
		Joins("JOIN projects ON projects.id = conversations.project_id AND projects.deleted_at IS NULL").
		Where("conversations.id = ? AND projects.owner_id = ?", conversationID, ownerID).
		First(&model).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	return conversationFromModel(model), nil
}

func (r *ConversationRepository) ListByProjectAndOwner(ctx context.Context, projectID, ownerID uuid.UUID) ([]domain.Conversation, error) {
	var models []conversationModel
	if err := r.db.WithContext(ctx).
		Table("conversations").
		Select("conversations.*").
		Joins("JOIN projects ON projects.id = conversations.project_id AND projects.deleted_at IS NULL").
		Where("conversations.project_id = ? AND projects.owner_id = ?", projectID, ownerID).
		Order("conversations.updated_at DESC").
		Find(&models).Error; err != nil {
		return nil, err
	}

	conversations := make([]domain.Conversation, 0, len(models))
	for _, model := range models {
		conversations = append(conversations, *conversationFromModel(model))
	}
	return conversations, nil
}

func (r *ConversationRepository) TouchLastMessage(ctx context.Context, conversationID uuid.UUID, at time.Time) error {
	result := r.db.WithContext(ctx).
		Model(&conversationModel{}).
		Where("id = ?", conversationID).
		Updates(map[string]any{
			"last_message_at": at,
			"updated_at":      at,
		})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return domain.ErrNotFound
	}
	return nil
}
