package handler

import (
	"errors"
	"net/http"
	"time"

	"backend/internal/application/project"
	"backend/internal/domain"
	"backend/internal/transport/http/middleware"
	"backend/internal/transport/http/response"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type ProjectHandler struct {
	projects *project.Service
}

type createProjectRequest struct {
	Name           string `json:"name" binding:"required"`
	Description    string `json:"description"`
	WorkingDir     string `json:"working_dir" binding:"required"`
	ProjectMode    string `json:"project_mode"`
	OutputLanguage string `json:"output_language"`
}

type updateProjectRequest struct {
	Name           *string `json:"name"`
	Description    *string `json:"description"`
	WorkingDir     *string `json:"working_dir"`
	ProjectMode    *string `json:"project_mode"`
	OutputLanguage *string `json:"output_language"`
}

type projectResponse struct {
	ID             uuid.UUID `json:"id"`
	Name           string    `json:"name"`
	Description    string    `json:"description"`
	WorkingDir     string    `json:"working_dir"`
	ProjectMode    string    `json:"project_mode"`
	Status         string    `json:"status"`
	OutputLanguage string    `json:"output_language"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

type projectListResponse struct {
	Items []projectResponse `json:"items"`
}

func NewProjectHandler(projects *project.Service) *ProjectHandler {
	return &ProjectHandler{projects: projects}
}

func (h *ProjectHandler) Create(c *gin.Context) {
	user, ok := middleware.CurrentUser(c)
	if !ok {
		response.Error(c, http.StatusUnauthorized, "auth.unauthorized", "missing user")
		return
	}

	var req createProjectRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Validation(c, err.Error())
		return
	}

	created, err := h.projects.Create(c.Request.Context(), user.ID, project.CreateInput{
		Name:           req.Name,
		Description:    req.Description,
		WorkingDir:     req.WorkingDir,
		ProjectMode:    req.ProjectMode,
		OutputLanguage: req.OutputLanguage,
	})
	if err != nil {
		writeProjectError(c, err)
		return
	}
	c.JSON(http.StatusCreated, toProjectResponse(created))
}

func (h *ProjectHandler) List(c *gin.Context) {
	user, ok := middleware.CurrentUser(c)
	if !ok {
		response.Error(c, http.StatusUnauthorized, "auth.unauthorized", "missing user")
		return
	}

	projects, err := h.projects.List(c.Request.Context(), user.ID)
	if err != nil {
		writeProjectError(c, err)
		return
	}

	items := make([]projectResponse, 0, len(projects))
	for i := range projects {
		items = append(items, toProjectResponse(&projects[i]))
	}
	c.JSON(http.StatusOK, projectListResponse{Items: items})
}

func (h *ProjectHandler) Get(c *gin.Context) {
	user, ok := middleware.CurrentUser(c)
	if !ok {
		response.Error(c, http.StatusUnauthorized, "auth.unauthorized", "missing user")
		return
	}

	projectID, ok := parseUUIDParam(c, "id")
	if !ok {
		return
	}

	found, err := h.projects.Get(c.Request.Context(), user.ID, projectID)
	if err != nil {
		writeProjectError(c, err)
		return
	}
	c.JSON(http.StatusOK, toProjectResponse(found))
}

func (h *ProjectHandler) Update(c *gin.Context) {
	user, ok := middleware.CurrentUser(c)
	if !ok {
		response.Error(c, http.StatusUnauthorized, "auth.unauthorized", "missing user")
		return
	}

	projectID, ok := parseUUIDParam(c, "id")
	if !ok {
		return
	}

	var req updateProjectRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Validation(c, err.Error())
		return
	}

	updated, err := h.projects.Update(c.Request.Context(), user.ID, projectID, project.UpdateInput{
		Name:           req.Name,
		Description:    req.Description,
		WorkingDir:     req.WorkingDir,
		ProjectMode:    req.ProjectMode,
		OutputLanguage: req.OutputLanguage,
	})
	if err != nil {
		writeProjectError(c, err)
		return
	}
	c.JSON(http.StatusOK, toProjectResponse(updated))
}

func parseUUIDParam(c *gin.Context, name string) (uuid.UUID, bool) {
	id, err := uuid.Parse(c.Param(name))
	if err != nil {
		response.Validation(c, "invalid "+name)
		return uuid.Nil, false
	}
	return id, true
}

func writeProjectError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, project.ErrValidation):
		response.Validation(c, err.Error())
	case errors.Is(err, domain.ErrNotFound):
		response.Error(c, http.StatusNotFound, "project.not_found", "project not found")
	case errors.Is(err, domain.ErrConflict):
		response.Error(c, http.StatusConflict, "project.name_exists", "project name already exists")
	default:
		response.Error(c, http.StatusInternalServerError, "internal_error", "project operation failed")
	}
}

func toProjectResponse(project *domain.Project) projectResponse {
	return projectResponse{
		ID:             project.ID,
		Name:           project.Name,
		Description:    project.Description,
		WorkingDir:     project.WorkingDir,
		ProjectMode:    project.ProjectMode,
		Status:         project.Status,
		OutputLanguage: project.OutputLanguage,
		CreatedAt:      project.CreatedAt,
		UpdatedAt:      project.UpdatedAt,
	}
}
