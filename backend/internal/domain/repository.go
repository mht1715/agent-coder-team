package domain

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
)

var (
	ErrNotFound           = errors.New("not found")
	ErrConflict           = errors.New("conflict")
	ErrRegistrationClosed = errors.New("registration closed")
)

type UserRepository interface {
	Create(ctx context.Context, user *User) error
	FindByUsername(ctx context.Context, username string) (*User, error)
	FindByID(ctx context.Context, id uuid.UUID) (*User, error)
	Count(ctx context.Context) (int64, error)
}

type SessionRepository interface {
	Create(ctx context.Context, session *Session) error
	FindByAccessTokenHash(ctx context.Context, hash string) (*Session, error)
	FindByRefreshTokenHash(ctx context.Context, hash string) (*Session, error)
	Rotate(ctx context.Context, id uuid.UUID, accessHash, refreshHash string, accessExpiresAt, refreshExpiresAt time.Time) error
	Revoke(ctx context.Context, id uuid.UUID) error
	Touch(ctx context.Context, id uuid.UUID) error
}

type ProjectRepository interface {
	Create(ctx context.Context, project *Project) error
	ListByOwner(ctx context.Context, ownerID uuid.UUID) ([]Project, error)
	FindByIDAndOwner(ctx context.Context, id, ownerID uuid.UUID) (*Project, error)
	Update(ctx context.Context, project *Project) error
}
