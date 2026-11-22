package handler

import (
	"net/http"

	workspaceapp "backend/internal/application/workspace"
	"backend/internal/transport/http/response"

	"github.com/gin-gonic/gin"
)

type WorkspaceHandler struct {
	workspaces *workspaceapp.Service
}

func NewWorkspaceHandler(workspaces *workspaceapp.Service) *WorkspaceHandler {
	return &WorkspaceHandler{workspaces: workspaces}
}

func (h *WorkspaceHandler) Browse(c *gin.Context) {
	result, err := h.workspaces.Browse(c.Query("path"))
	if err != nil {
		response.Validation(c, err.Error())
		return
	}
	c.JSON(http.StatusOK, result)
}
