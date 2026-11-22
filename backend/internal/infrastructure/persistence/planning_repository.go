package persistence

import (
	"context"
	"errors"
	"time"

	"backend/internal/domain"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type RequirementRepository struct {
	db *gorm.DB
}

func NewRequirementRepository(db *gorm.DB) *RequirementRepository {
	return &RequirementRepository{db: db}
}

func (r *RequirementRepository) Upsert(ctx context.Context, requirement *domain.RequirementDocument) error {
	model := requirementToModel(requirement)
	return r.db.WithContext(ctx).
		Clauses(clause.OnConflict{
			Columns: []clause.Column{{Name: "project_id"}},
			DoUpdates: clause.AssignmentColumns([]string{
				"title",
				"content_md",
				"status",
				"version",
				"generated_by",
				"approved_at",
				"updated_at",
			}),
		}).
		Create(&model).Error
}

func (r *RequirementRepository) FindByProjectID(ctx context.Context, projectID uuid.UUID) (*domain.RequirementDocument, error) {
	var model requirementModel
	if err := r.db.WithContext(ctx).First(&model, "project_id = ?", projectID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	return requirementFromModel(model), nil
}

type TaskRepository struct {
	db *gorm.DB
}

func NewTaskRepository(db *gorm.DB) *TaskRepository {
	return &TaskRepository{db: db}
}

func (r *TaskRepository) ReplacePlan(ctx context.Context, projectID uuid.UUID, tasks []domain.Task) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("project_id = ?", projectID).Delete(&taskModel{}).Error; err != nil {
			return err
		}

		if len(tasks) == 0 {
			return nil
		}

		models := make([]taskModel, 0, len(tasks))
		dependencies := make([]taskDependencyModel, 0)
		for i := range tasks {
			model := taskToModel(&tasks[i])
			models = append(models, model)
			for _, dependencyID := range tasks[i].Dependencies {
				dependencies = append(dependencies, taskDependencyModel{
					TaskID:          tasks[i].ID,
					DependsOnTaskID: dependencyID,
				})
			}
		}
		if err := tx.Create(&models).Error; err != nil {
			return err
		}
		if len(dependencies) > 0 {
			if err := tx.Create(&dependencies).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

func (r *TaskRepository) ListByProject(ctx context.Context, projectID uuid.UUID) ([]domain.Task, error) {
	var models []taskModel
	if err := r.db.WithContext(ctx).
		Where("project_id = ?", projectID).
		Order("position ASC, created_at ASC").
		Find(&models).Error; err != nil {
		return nil, err
	}
	if len(models) == 0 {
		return []domain.Task{}, nil
	}

	ids := make([]uuid.UUID, 0, len(models))
	for i := range models {
		ids = append(ids, models[i].ID)
	}
	var dependencyModels []taskDependencyModel
	if err := r.db.WithContext(ctx).Where("task_id IN ?", ids).Find(&dependencyModels).Error; err != nil {
		return nil, err
	}
	dependencies := make(map[uuid.UUID][]uuid.UUID, len(models))
	for _, dependency := range dependencyModels {
		dependencies[dependency.TaskID] = append(dependencies[dependency.TaskID], dependency.DependsOnTaskID)
	}

	tasks := make([]domain.Task, 0, len(models))
	for _, model := range models {
		tasks = append(tasks, *taskFromModel(model, dependencies[model.ID]))
	}
	return tasks, nil
}

func (r *TaskRepository) FindByIDAndOwner(ctx context.Context, taskID, ownerID uuid.UUID) (*domain.Task, error) {
	var model taskModel
	if err := r.db.WithContext(ctx).
		Table("tasks").
		Select("tasks.*").
		Joins("JOIN projects ON projects.id = tasks.project_id AND projects.deleted_at IS NULL").
		Where("tasks.id = ? AND projects.owner_id = ?", taskID, ownerID).
		First(&model).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}

	var dependencyModels []taskDependencyModel
	if err := r.db.WithContext(ctx).Where("task_id = ?", taskID).Find(&dependencyModels).Error; err != nil {
		return nil, err
	}
	dependencies := make([]uuid.UUID, 0, len(dependencyModels))
	for _, dependency := range dependencyModels {
		dependencies = append(dependencies, dependency.DependsOnTaskID)
	}
	return taskFromModel(model, dependencies), nil
}

func (r *TaskRepository) Update(ctx context.Context, task *domain.Task) error {
	result := r.db.WithContext(ctx).
		Model(&taskModel{}).
		Where("id = ? AND project_id = ?", task.ID, task.ProjectID).
		Updates(map[string]any{
			"title":               task.Title,
			"description":         task.Description,
			"acceptance_criteria": task.AcceptanceCriteria,
			"agent_role":          task.AgentRole,
			"priority":            task.Priority,
			"status":              task.Status,
			"position":            task.Position,
			"started_at":          task.StartedAt,
			"completed_at":        task.CompletedAt,
			"updated_at":          task.UpdatedAt,
		})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return domain.ErrNotFound
	}
	return nil
}

type ApprovalRepository struct {
	db *gorm.DB
}

func NewApprovalRepository(db *gorm.DB) *ApprovalRepository {
	return &ApprovalRepository{db: db}
}

func (r *ApprovalRepository) Create(ctx context.Context, approval *domain.Approval) error {
	return r.db.WithContext(ctx).Create(ptr(approvalToModel(approval))).Error
}

func (r *ApprovalRepository) Update(ctx context.Context, approval *domain.Approval) error {
	result := r.db.WithContext(ctx).
		Model(&approvalModel{}).
		Where("id = ? AND project_id = ?", approval.ID, approval.ProjectID).
		Updates(map[string]any{
			"task_id":       approval.TaskID,
			"run_id":        approval.RunID,
			"status":        approval.Status,
			"decision_note": approval.DecisionNote,
			"decided_by":    approval.DecidedBy,
			"decided_at":    approval.DecidedAt,
			"updated_at":    approval.UpdatedAt,
		})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (r *ApprovalRepository) FindByIDAndOwner(ctx context.Context, approvalID, ownerID uuid.UUID) (*domain.Approval, error) {
	var model approvalModel
	if err := r.db.WithContext(ctx).
		Table("approvals").
		Select("approvals.*").
		Joins("JOIN projects ON projects.id = approvals.project_id AND projects.deleted_at IS NULL").
		Where("approvals.id = ? AND projects.owner_id = ?", approvalID, ownerID).
		First(&model).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	return approvalFromModel(model), nil
}

func (r *ApprovalRepository) ListByProject(ctx context.Context, projectID uuid.UUID, status string) ([]domain.Approval, error) {
	var models []approvalModel
	query := r.db.WithContext(ctx).Where("project_id = ?", projectID)
	if status != "" {
		query = query.Where("status = ?", status)
	}
	if err := query.Order("created_at DESC").Find(&models).Error; err != nil {
		return nil, err
	}
	approvals := make([]domain.Approval, 0, len(models))
	for _, model := range models {
		approvals = append(approvals, *approvalFromModel(model))
	}
	return approvals, nil
}

type AgentRunRepository struct {
	db *gorm.DB
}

func NewAgentRunRepository(db *gorm.DB) *AgentRunRepository {
	return &AgentRunRepository{db: db}
}

func (r *AgentRunRepository) Create(ctx context.Context, run *domain.AgentRun) error {
	return r.db.WithContext(ctx).Create(ptr(agentRunToModel(run))).Error
}

func (r *AgentRunRepository) Update(ctx context.Context, run *domain.AgentRun) error {
	run.UpdatedAt = time.Now().UTC()
	result := r.db.WithContext(ctx).
		Model(&agentRunModel{}).
		Where("id = ? AND project_id = ?", run.ID, run.ProjectID).
		Updates(map[string]any{
			"status":         run.Status,
			"output_summary": run.OutputSummary,
			"logs":           agentRunToModel(run).Logs,
			"started_at":     run.StartedAt,
			"finished_at":    run.FinishedAt,
			"updated_at":     run.UpdatedAt,
		})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (r *AgentRunRepository) FindByIDAndOwner(ctx context.Context, runID, ownerID uuid.UUID) (*domain.AgentRun, error) {
	var model agentRunModel
	if err := r.db.WithContext(ctx).
		Table("agent_runs").
		Select("agent_runs.*").
		Joins("JOIN projects ON projects.id = agent_runs.project_id AND projects.deleted_at IS NULL").
		Where("agent_runs.id = ? AND projects.owner_id = ?", runID, ownerID).
		First(&model).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	return agentRunFromModel(model), nil
}

func (r *AgentRunRepository) ListByProject(ctx context.Context, projectID uuid.UUID) ([]domain.AgentRun, error) {
	var models []agentRunModel
	if err := r.db.WithContext(ctx).
		Where("project_id = ?", projectID).
		Order("created_at DESC").
		Limit(200).
		Find(&models).Error; err != nil {
		return nil, err
	}
	runs := make([]domain.AgentRun, 0, len(models))
	for _, model := range models {
		runs = append(runs, *agentRunFromModel(model))
	}
	return runs, nil
}
