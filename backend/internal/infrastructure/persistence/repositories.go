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
	Profiles      domain.ProjectProfileRepository
	Requirements  domain.RequirementRepository
	Tasks         domain.TaskRepository
	Approvals     domain.ApprovalRepository
	Runs          domain.AgentRunRepository
}

func NewRepositories(db *gorm.DB) *Repositories {
	return &Repositories{
		Users:         NewUserRepository(db),
		Sessions:      NewSessionRepository(db),
		Projects:      NewProjectRepository(db),
		Conversations: NewConversationRepository(db),
		Messages:      NewMessageRepository(db),
		Profiles:      NewProjectProfileRepository(db),
		Requirements:  NewRequirementRepository(db),
		Tasks:         NewTaskRepository(db),
		Approvals:     NewApprovalRepository(db),
		Runs:          NewAgentRunRepository(db),
	}
}
