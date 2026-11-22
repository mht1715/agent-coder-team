package persistence

import (
	"backend/internal/domain"

	"gorm.io/gorm"
)

type Repositories struct {
	Users         domain.UserRepository
	Sessions      domain.SessionRepository
	Projects      domain.ProjectRepository
	Conversations domain.ConversationRepository
	Messages      domain.MessageRepository
}

func NewRepositories(db *gorm.DB) *Repositories {
	return &Repositories{
		Users:         NewUserRepository(db),
		Sessions:      NewSessionRepository(db),
		Projects:      NewProjectRepository(db),
		Conversations: NewConversationRepository(db),
		Messages:      NewMessageRepository(db),
	}
}
