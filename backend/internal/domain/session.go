package domain

import (
	"time"

	"github.com/google/uuid"
)

type Session struct {
	ID               uuid.UUID
	UserID           uuid.UUID
	AccessTokenHash  string
	RefreshTokenHash string
	AccessExpiresAt  time.Time
	RefreshExpiresAt time.Time
	RevokedAt        *time.Time
	UserAgent        string
	ClientIP         string
	CreatedAt        time.Time
	LastUsedAt       *time.Time
}

func (s *Session) IsAccessValid(now time.Time) bool {
	return s.RevokedAt == nil && now.Before(s.AccessExpiresAt)
}

func (s *Session) IsRefreshValid(now time.Time) bool {
	return s.RevokedAt == nil && now.Before(s.RefreshExpiresAt)
}
