package persistence

import (
	"time"

	"backend/internal/domain"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type userModel struct {
	ID           uuid.UUID      `gorm:"type:uuid;primaryKey"`
	Username     string         `gorm:"size:64;uniqueIndex;not null"`
	PasswordHash string         `gorm:"not null"`
	DisplayName  string         `gorm:"size:128;not null;default:''"`
	Status       string         `gorm:"size:32;not null;default:'active'"`
	CreatedAt    time.Time      `gorm:"not null"`
	UpdatedAt    time.Time      `gorm:"not null"`
	DeletedAt    gorm.DeletedAt `gorm:"index"`
}

func (userModel) TableName() string { return "users" }

type sessionModel struct {
	ID               uuid.UUID `gorm:"type:uuid;primaryKey"`
	UserID           uuid.UUID `gorm:"type:uuid;not null;index"`
	AccessTokenHash  string    `gorm:"size:64;uniqueIndex;not null"`
	RefreshTokenHash string    `gorm:"size:64;uniqueIndex;not null"`
	AccessExpiresAt  time.Time `gorm:"not null"`
	RefreshExpiresAt time.Time `gorm:"not null"`
	RevokedAt        *time.Time
	UserAgent        string    `gorm:"not null;default:''"`
	ClientIP         string    `gorm:"size:64;not null;default:''"`
	CreatedAt        time.Time `gorm:"not null"`
	LastUsedAt       *time.Time
}

func (sessionModel) TableName() string { return "sessions" }

type projectModel struct {
	ID             uuid.UUID      `gorm:"type:uuid;primaryKey"`
	OwnerID        uuid.UUID      `gorm:"type:uuid;not null;index"`
	Name           string         `gorm:"size:128;not null"`
	Description    string         `gorm:"not null;default:''"`
	WorkingDir     string         `gorm:"not null;default:''"`
	ProjectMode    string         `gorm:"size:16;not null;default:'new'"`
	Status         string         `gorm:"size:32;not null;default:'draft'"`
	OutputLanguage string         `gorm:"size:16;not null;default:'zh-CN'"`
	CreatedAt      time.Time      `gorm:"not null"`
	UpdatedAt      time.Time      `gorm:"not null"`
	DeletedAt      gorm.DeletedAt `gorm:"index"`
}

func (projectModel) TableName() string { return "projects" }

func userFromModel(m userModel) *domain.User {
	return &domain.User{
		ID:           m.ID,
		Username:     m.Username,
		PasswordHash: m.PasswordHash,
		DisplayName:  m.DisplayName,
		Status:       m.Status,
		CreatedAt:    m.CreatedAt,
		UpdatedAt:    m.UpdatedAt,
	}
}

func userToModel(u *domain.User) userModel {
	return userModel{
		ID:           u.ID,
		Username:     u.Username,
		PasswordHash: u.PasswordHash,
		DisplayName:  u.DisplayName,
		Status:       u.Status,
		CreatedAt:    u.CreatedAt,
		UpdatedAt:    u.UpdatedAt,
	}
}

func sessionFromModel(m sessionModel) *domain.Session {
	return &domain.Session{
		ID:               m.ID,
		UserID:           m.UserID,
		AccessTokenHash:  m.AccessTokenHash,
		RefreshTokenHash: m.RefreshTokenHash,
		AccessExpiresAt:  m.AccessExpiresAt,
		RefreshExpiresAt: m.RefreshExpiresAt,
		RevokedAt:        m.RevokedAt,
		UserAgent:        m.UserAgent,
		ClientIP:         m.ClientIP,
		CreatedAt:        m.CreatedAt,
		LastUsedAt:       m.LastUsedAt,
	}
}

func sessionToModel(s *domain.Session) sessionModel {
	return sessionModel{
		ID:               s.ID,
		UserID:           s.UserID,
		AccessTokenHash:  s.AccessTokenHash,
		RefreshTokenHash: s.RefreshTokenHash,
		AccessExpiresAt:  s.AccessExpiresAt,
		RefreshExpiresAt: s.RefreshExpiresAt,
		RevokedAt:        s.RevokedAt,
		UserAgent:        s.UserAgent,
		ClientIP:         s.ClientIP,
		CreatedAt:        s.CreatedAt,
		LastUsedAt:       s.LastUsedAt,
	}
}

func projectFromModel(m projectModel) *domain.Project {
	return &domain.Project{
		ID:             m.ID,
		OwnerID:        m.OwnerID,
		Name:           m.Name,
		Description:    m.Description,
		WorkingDir:     m.WorkingDir,
		ProjectMode:    m.ProjectMode,
		Status:         m.Status,
		OutputLanguage: m.OutputLanguage,
		CreatedAt:      m.CreatedAt,
		UpdatedAt:      m.UpdatedAt,
	}
}

func projectToModel(p *domain.Project) projectModel {
	return projectModel{
		ID:             p.ID,
		OwnerID:        p.OwnerID,
		Name:           p.Name,
		Description:    p.Description,
		WorkingDir:     p.WorkingDir,
		ProjectMode:    p.ProjectMode,
		Status:         p.Status,
		OutputLanguage: p.OutputLanguage,
		CreatedAt:      p.CreatedAt,
		UpdatedAt:      p.UpdatedAt,
	}
}
