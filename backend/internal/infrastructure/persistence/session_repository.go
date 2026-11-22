package persistence

import (
	"context"
	"errors"
	"time"

	"backend/internal/domain"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type SessionRepository struct {
	db *gorm.DB
}

func NewSessionRepository(db *gorm.DB) *SessionRepository {
	return &SessionRepository{db: db}
}

func (r *SessionRepository) Create(ctx context.Context, session *domain.Session) error {
	return r.db.WithContext(ctx).Create(ptr(sessionToModel(session))).Error
}

func (r *SessionRepository) FindByAccessTokenHash(ctx context.Context, hash string) (*domain.Session, error) {
	return r.findByHash(ctx, "access_token_hash", hash)
}

func (r *SessionRepository) FindByRefreshTokenHash(ctx context.Context, hash string) (*domain.Session, error) {
	return r.findByHash(ctx, "refresh_token_hash", hash)
}

func (r *SessionRepository) findByHash(ctx context.Context, column, hash string) (*domain.Session, error) {
	var model sessionModel
	if err := r.db.WithContext(ctx).Where(column+" = ?", hash).First(&model).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	return sessionFromModel(model), nil
}

func (r *SessionRepository) Rotate(
	ctx context.Context,
	id uuid.UUID,
	accessHash, refreshHash string,
	accessExpiresAt, refreshExpiresAt time.Time,
) error {
	result := r.db.WithContext(ctx).
		Model(&sessionModel{}).
		Where("id = ? AND revoked_at IS NULL", id).
		Updates(map[string]any{
			"access_token_hash":  accessHash,
			"refresh_token_hash": refreshHash,
			"access_expires_at":  accessExpiresAt,
			"refresh_expires_at": refreshExpiresAt,
			"last_used_at":       time.Now().UTC(),
		})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (r *SessionRepository) Revoke(ctx context.Context, id uuid.UUID) error {
	result := r.db.WithContext(ctx).
		Model(&sessionModel{}).
		Where("id = ? AND revoked_at IS NULL", id).
		Update("revoked_at", time.Now().UTC())
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (r *SessionRepository) Touch(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).
		Model(&sessionModel{}).
		Where("id = ?", id).
		Update("last_used_at", time.Now().UTC()).Error
}
