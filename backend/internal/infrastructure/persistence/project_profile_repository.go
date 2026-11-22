package persistence

import (
	"context"
	"errors"

	"backend/internal/domain"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type ProjectProfileRepository struct {
	db *gorm.DB
}

func NewProjectProfileRepository(db *gorm.DB) *ProjectProfileRepository {
	return &ProjectProfileRepository{db: db}
}

func (r *ProjectProfileRepository) Upsert(ctx context.Context, profile *domain.ProjectProfile) error {
	model := projectProfileToModel(profile)
	return r.db.WithContext(ctx).
		Clauses(clause.OnConflict{
			Columns: []clause.Column{{Name: "project_id"}},
			DoUpdates: clause.AssignmentColumns([]string{
				"primary_language",
				"languages",
				"frameworks",
				"package_manager",
				"build_command",
				"test_command",
				"format_command",
				"file_count",
				"top_level",
				"truncated",
				"detected_at",
				"updated_at",
			}),
		}).
		Create(&model).Error
}

func (r *ProjectProfileRepository) FindByProjectID(ctx context.Context, projectID uuid.UUID) (*domain.ProjectProfile, error) {
	var model projectProfileModel
	if err := r.db.WithContext(ctx).First(&model, "project_id = ?", projectID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	return projectProfileFromModel(model), nil
}
