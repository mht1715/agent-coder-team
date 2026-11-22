package persistence

import (
	"encoding/json"
	"time"

	"backend/internal/domain"

	"github.com/google/uuid"
	"gorm.io/datatypes"
)

type requirementModel struct {
	ID          uuid.UUID `gorm:"type:uuid;primaryKey"`
	ProjectID   uuid.UUID `gorm:"type:uuid;not null;uniqueIndex"`
	Title       string    `gorm:"size:255;not null"`
	ContentMD   string    `gorm:"column:content_md;not null"`
	Status      string    `gorm:"size:32;not null;default:'draft'"`
	Version     int       `gorm:"not null;default:1"`
	GeneratedBy string    `gorm:"size:32;not null;default:'main_agent'"`
	ApprovedAt  *time.Time
	CreatedAt   time.Time `gorm:"not null"`
	UpdatedAt   time.Time `gorm:"not null"`
}

func (requirementModel) TableName() string { return "requirement_documents" }

type taskModel struct {
	ID                 uuid.UUID `gorm:"type:uuid;primaryKey"`
	ProjectID          uuid.UUID `gorm:"type:uuid;not null;index"`
	TaskKey            string    `gorm:"size:32;not null"`
	Title              string    `gorm:"size:255;not null"`
	Description        string    `gorm:"not null;default:''"`
	AcceptanceCriteria string    `gorm:"not null;default:''"`
	AgentRole          string    `gorm:"size:64;not null;default:''"`
	Priority           string    `gorm:"size:16;not null;default:'medium'"`
	Status             string    `gorm:"size:32;not null;default:'backlog'"`
	Position           int       `gorm:"not null;default:0"`
	ExecutionCommand   string    `gorm:"not null;default:''"`
	ExecutionImage     string    `gorm:"size:255;not null;default:''"`
	StartedAt          *time.Time
	CompletedAt        *time.Time
	CreatedAt          time.Time `gorm:"not null"`
	UpdatedAt          time.Time `gorm:"not null"`
}

func (taskModel) TableName() string { return "tasks" }

type taskDependencyModel struct {
	TaskID          uuid.UUID `gorm:"type:uuid;primaryKey"`
	DependsOnTaskID uuid.UUID `gorm:"type:uuid;primaryKey"`
}

func (taskDependencyModel) TableName() string { return "task_dependencies" }

type approvalModel struct {
	ID              uuid.UUID  `gorm:"type:uuid;primaryKey"`
	ProjectID       uuid.UUID  `gorm:"type:uuid;not null;index"`
	TaskID          *uuid.UUID `gorm:"type:uuid"`
	RunID           *uuid.UUID `gorm:"type:uuid"`
	Kind            string     `gorm:"size:32;not null"`
	Status          string     `gorm:"size:32;not null;default:'pending'"`
	Title           string     `gorm:"size:255;not null"`
	Description     string     `gorm:"not null;default:''"`
	RequestedByType string     `gorm:"size:32;not null;default:'main_agent'"`
	DecidedBy       *uuid.UUID `gorm:"type:uuid"`
	DecisionNote    string     `gorm:"not null;default:''"`
	DecidedAt       *time.Time
	CreatedAt       time.Time `gorm:"not null"`
	UpdatedAt       time.Time `gorm:"not null"`
}

func (approvalModel) TableName() string { return "approvals" }

type agentRunModel struct {
	ID             uuid.UUID  `gorm:"type:uuid;primaryKey"`
	ProjectID      uuid.UUID  `gorm:"type:uuid;not null;index"`
	TaskID         *uuid.UUID `gorm:"type:uuid"`
	ConversationID *uuid.UUID `gorm:"type:uuid"`
	Kind           string     `gorm:"size:32;not null"`
	Status         string     `gorm:"size:32;not null;default:'queued'"`
	Title          string     `gorm:"size:255;not null"`
	InputSummary   string     `gorm:"not null;default:''"`
	OutputSummary  string     `gorm:"not null;default:''"`
	Executor       string     `gorm:"size:32;not null;default:''"`
	ExitCode       *int
	ChangedFiles   datatypes.JSON `gorm:"type:jsonb;not null;default:'[]'"`
	Logs           datatypes.JSON `gorm:"type:jsonb;not null;default:'[]'"`
	StartedAt      *time.Time
	FinishedAt     *time.Time
	CreatedAt      time.Time `gorm:"not null"`
	UpdatedAt      time.Time `gorm:"not null"`
}

func (agentRunModel) TableName() string { return "agent_runs" }

func requirementFromModel(m requirementModel) *domain.RequirementDocument {
	return &domain.RequirementDocument{
		ID:          m.ID,
		ProjectID:   m.ProjectID,
		Title:       m.Title,
		ContentMD:   m.ContentMD,
		Status:      m.Status,
		Version:     m.Version,
		GeneratedBy: m.GeneratedBy,
		ApprovedAt:  m.ApprovedAt,
		CreatedAt:   m.CreatedAt,
		UpdatedAt:   m.UpdatedAt,
	}
}

func requirementToModel(r *domain.RequirementDocument) requirementModel {
	return requirementModel{
		ID:          r.ID,
		ProjectID:   r.ProjectID,
		Title:       r.Title,
		ContentMD:   r.ContentMD,
		Status:      r.Status,
		Version:     r.Version,
		GeneratedBy: r.GeneratedBy,
		ApprovedAt:  r.ApprovedAt,
		CreatedAt:   r.CreatedAt,
		UpdatedAt:   r.UpdatedAt,
	}
}

func taskFromModel(m taskModel, dependencies []uuid.UUID) *domain.Task {
	if dependencies == nil {
		dependencies = []uuid.UUID{}
	}
	return &domain.Task{
		ID:                 m.ID,
		ProjectID:          m.ProjectID,
		TaskKey:            m.TaskKey,
		Title:              m.Title,
		Description:        m.Description,
		AcceptanceCriteria: m.AcceptanceCriteria,
		AgentRole:          m.AgentRole,
		Priority:           m.Priority,
		Status:             m.Status,
		Position:           m.Position,
		ExecutionCommand:   m.ExecutionCommand,
		ExecutionImage:     m.ExecutionImage,
		Dependencies:       dependencies,
		StartedAt:          m.StartedAt,
		CompletedAt:        m.CompletedAt,
		CreatedAt:          m.CreatedAt,
		UpdatedAt:          m.UpdatedAt,
	}
}

func taskToModel(t *domain.Task) taskModel {
	return taskModel{
		ID:                 t.ID,
		ProjectID:          t.ProjectID,
		TaskKey:            t.TaskKey,
		Title:              t.Title,
		Description:        t.Description,
		AcceptanceCriteria: t.AcceptanceCriteria,
		AgentRole:          t.AgentRole,
		Priority:           t.Priority,
		Status:             t.Status,
		Position:           t.Position,
		ExecutionCommand:   t.ExecutionCommand,
		ExecutionImage:     t.ExecutionImage,
		StartedAt:          t.StartedAt,
		CompletedAt:        t.CompletedAt,
		CreatedAt:          t.CreatedAt,
		UpdatedAt:          t.UpdatedAt,
	}
}

func approvalFromModel(m approvalModel) *domain.Approval {
	return &domain.Approval{
		ID:              m.ID,
		ProjectID:       m.ProjectID,
		TaskID:          m.TaskID,
		RunID:           m.RunID,
		Kind:            m.Kind,
		Status:          m.Status,
		Title:           m.Title,
		Description:     m.Description,
		RequestedByType: m.RequestedByType,
		DecidedBy:       m.DecidedBy,
		DecisionNote:    m.DecisionNote,
		DecidedAt:       m.DecidedAt,
		CreatedAt:       m.CreatedAt,
		UpdatedAt:       m.UpdatedAt,
	}
}

func approvalToModel(a *domain.Approval) approvalModel {
	return approvalModel{
		ID:              a.ID,
		ProjectID:       a.ProjectID,
		TaskID:          a.TaskID,
		RunID:           a.RunID,
		Kind:            a.Kind,
		Status:          a.Status,
		Title:           a.Title,
		Description:     a.Description,
		RequestedByType: a.RequestedByType,
		DecidedBy:       a.DecidedBy,
		DecisionNote:    a.DecisionNote,
		DecidedAt:       a.DecidedAt,
		CreatedAt:       a.CreatedAt,
		UpdatedAt:       a.UpdatedAt,
	}
}

func agentRunFromModel(m agentRunModel) *domain.AgentRun {
	logs := []domain.RunLog{}
	if len(m.Logs) > 0 {
		_ = json.Unmarshal(m.Logs, &logs)
	}
	if logs == nil {
		logs = []domain.RunLog{}
	}
	changedFiles := []string{}
	if len(m.ChangedFiles) > 0 {
		_ = json.Unmarshal(m.ChangedFiles, &changedFiles)
	}
	if changedFiles == nil {
		changedFiles = []string{}
	}
	return &domain.AgentRun{
		ID:             m.ID,
		ProjectID:      m.ProjectID,
		TaskID:         m.TaskID,
		ConversationID: m.ConversationID,
		Kind:           m.Kind,
		Status:         m.Status,
		Title:          m.Title,
		InputSummary:   m.InputSummary,
		OutputSummary:  m.OutputSummary,
		Executor:       m.Executor,
		ExitCode:       m.ExitCode,
		ChangedFiles:   changedFiles,
		Logs:           logs,
		StartedAt:      m.StartedAt,
		FinishedAt:     m.FinishedAt,
		CreatedAt:      m.CreatedAt,
		UpdatedAt:      m.UpdatedAt,
	}
}

func agentRunToModel(r *domain.AgentRun) agentRunModel {
	logs := r.Logs
	if logs == nil {
		logs = []domain.RunLog{}
	}
	changedFiles := r.ChangedFiles
	if changedFiles == nil {
		changedFiles = []string{}
	}
	raw, _ := json.Marshal(logs)
	changedRaw, _ := json.Marshal(changedFiles)
	return agentRunModel{
		ID:             r.ID,
		ProjectID:      r.ProjectID,
		TaskID:         r.TaskID,
		ConversationID: r.ConversationID,
		Kind:           r.Kind,
		Status:         r.Status,
		Title:          r.Title,
		InputSummary:   r.InputSummary,
		OutputSummary:  r.OutputSummary,
		Executor:       r.Executor,
		ExitCode:       r.ExitCode,
		ChangedFiles:   datatypes.JSON(changedRaw),
		Logs:           datatypes.JSON(raw),
		StartedAt:      r.StartedAt,
		FinishedAt:     r.FinishedAt,
		CreatedAt:      r.CreatedAt,
		UpdatedAt:      r.UpdatedAt,
	}
}
