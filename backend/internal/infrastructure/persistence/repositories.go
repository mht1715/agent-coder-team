package persistence

import (
	"backend/internal/domain"

	"gorm.io/gorm"
)

type Repositories struct {
	Users    domain.UserRepository
	Sessions domain.SessionRepository
	Projects domain.ProjectRepository
}

func NewRepositories(db *gorm.DB) *Repositories {
	return &Repositories{
		Users:    NewUserRepository(db),
		Sessions: NewSessionRepository(db),
		Projects: NewProjectRepository(db),
	}
}
