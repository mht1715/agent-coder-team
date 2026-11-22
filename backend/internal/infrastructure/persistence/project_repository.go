package persistence

import (
	"context"
	"errors"

	"backend/internal/domain"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type ProjectRepository struct {
	db *gorm.DB
}

func NewProjectRepository(db *gorm.DB) *ProjectRepository {
	return &ProjectRepository{db: db}
}

func (r *ProjectRepository) Create(ctx context.Context, project *domain.Project) error {
	if err := r.db.WithContext(ctx).Create(ptr(projectToModel(project))).Error; err != nil {
		if errors.Is(err, gorm.ErrDuplicatedKey) {
			return domain.ErrConflict
		}
		return err
	}
	return nil
}

func (r *ProjectRepository) ListByOwner(ctx context.Context, ownerID uuid.UUID) ([]domain.Project, error) {
	var models []projectModel
	if err := r.db.WithContext(ctx).
		Where("owner_id = ?", ownerID).
		Order("updated_at DESC").
		Find(&models).Error; err != nil {
		return nil, err
	}

	projects := make([]domain.Project, 0, len(models))
	for _, model := range models {
		projects = append(projects, *projectFromModel(model))
	}
	return projects, nil
}

func (r *ProjectRepository) FindByIDAndOwner(ctx context.Context, id, ownerID uuid.UUID) (*domain.Project, error) {
	var model projectModel
	if err := r.db.WithContext(ctx).
		Where("id = ? AND owner_id = ?", id, ownerID).
		First(&model).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	return projectFromModel(model), nil
}

func (r *ProjectRepository) Update(ctx context.Context, project *domain.Project) error {
	result := r.db.WithContext(ctx).
		Model(&projectModel{}).
		Where("id = ? AND owner_id = ?", project.ID, project.OwnerID).
		Updates(map[string]any{
			"name":            project.Name,
			"description":     project.Description,
			"working_dir":     project.WorkingDir,
			"project_mode":    project.ProjectMode,
			"status":          project.Status,
			"output_language": project.OutputLanguage,
			"updated_at":      project.UpdatedAt,
		})
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrDuplicatedKey) {
			return domain.ErrConflict
		}
		return result.Error
	}
	if result.RowsAffected == 0 {
		return domain.ErrNotFound
	}
	return nil
}
