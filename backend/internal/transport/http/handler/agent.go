package handler

import (
	"errors"
	"net/http"
	"strings"

	agentapp "backend/internal/application/agent"
	"backend/internal/domain"
	"backend/internal/transport/http/middleware"
	"backend/internal/transport/http/response"

	"github.com/gin-gonic/gin"
)

type AgentHandler struct {
	agent *agentapp.Service
}

type updateRequirementRequest struct {
	Title     string `json:"title" binding:"required"`
	ContentMD string `json:"content_md" binding:"required"`
}

type generatePlanRequest struct {
	Replace bool `json:"replace"`
}

type transitionTaskRequest struct {
	Status string `json:"status" binding:"required"`
	Note   string `json:"note"`
}

type decideApprovalRequest struct {
	Decision string `json:"decision" binding:"required"`
	Note     string `json:"note"`
}

type finishRunRequest struct {
	Status        string `json:"status" binding:"required"`
	OutputSummary string `json:"output_summary"`
}

type taskListResponse struct {
	Items []domain.Task `json:"items"`
}
type approvalListResponse struct {
	Items []domain.Approval `json:"items"`
}
type runListResponse struct {
	Items []domain.AgentRun `json:"items"`
}
type generatePlanResponse struct {
	Tasks    []domain.Task    `json:"tasks"`
	Approval *domain.Approval `json:"approval,omitempty"`
}

func NewAgentHandler(agent *agentapp.Service) *AgentHandler {
	return &AgentHandler{agent: agent}
}

func (h *AgentHandler) GetRequirement(c *gin.Context) {
	user, ok := middleware.CurrentUser(c)
	if !ok {
		response.Error(c, http.StatusUnauthorized, "auth.unauthorized", "missing user")
		return
	}
	projectID, ok := parseUUIDParam(c, "id")
	if !ok {
		return
	}
	requirement, err := h.agent.GetRequirement(c.Request.Context(), user.ID, projectID)
	if err != nil {
		writeAgentError(c, err)
		return
	}
	c.JSON(http.StatusOK, requirement)
}

func (h *AgentHandler) GenerateRequirement(c *gin.Context) {
	user, ok := middleware.CurrentUser(c)
	if !ok {
		response.Error(c, http.StatusUnauthorized, "auth.unauthorized", "missing user")
		return
	}
	projectID, ok := parseUUIDParam(c, "id")
	if !ok {
		return
	}
	requirement, err := h.agent.GenerateRequirement(c.Request.Context(), user.ID, projectID)
	if err != nil {
		writeAgentError(c, err)
		return
	}
	c.JSON(http.StatusCreated, requirement)
}

func (h *AgentHandler) UpdateRequirement(c *gin.Context) {
	user, ok := middleware.CurrentUser(c)
	if !ok {
		response.Error(c, http.StatusUnauthorized, "auth.unauthorized", "missing user")
		return
	}
	projectID, ok := parseUUIDParam(c, "id")
	if !ok {
		return
	}
	var req updateRequirementRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Validation(c, err.Error())
		return
	}
	requirement, err := h.agent.UpdateRequirement(c.Request.Context(), user.ID, projectID, req.Title, req.ContentMD)
	if err != nil {
		writeAgentError(c, err)
		return
	}
	c.JSON(http.StatusOK, requirement)
}

func (h *AgentHandler) ApproveRequirement(c *gin.Context) {
	user, ok := middleware.CurrentUser(c)
	if !ok {
		response.Error(c, http.StatusUnauthorized, "auth.unauthorized", "missing user")
		return
	}
	projectID, ok := parseUUIDParam(c, "id")
	if !ok {
		return
	}
	requirement, err := h.agent.ApproveRequirement(c.Request.Context(), user.ID, projectID)
	if err != nil {
		writeAgentError(c, err)
		return
	}
	c.JSON(http.StatusOK, requirement)
}

func (h *AgentHandler) ListTasks(c *gin.Context) {
	user, ok := middleware.CurrentUser(c)
	if !ok {
		response.Error(c, http.StatusUnauthorized, "auth.unauthorized", "missing user")
		return
	}
	projectID, ok := parseUUIDParam(c, "id")
	if !ok {
		return
	}
	tasks, err := h.agent.ListTasks(c.Request.Context(), user.ID, projectID)
	if err != nil {
		writeAgentError(c, err)
		return
	}
	c.JSON(http.StatusOK, taskListResponse{Items: tasks})
}

func (h *AgentHandler) GeneratePlan(c *gin.Context) {
	user, ok := middleware.CurrentUser(c)
	if !ok {
		response.Error(c, http.StatusUnauthorized, "auth.unauthorized", "missing user")
		return
	}
	projectID, ok := parseUUIDParam(c, "id")
	if !ok {
		return
	}
	var req generatePlanRequest
	if c.Request.ContentLength > 0 {
		if err := c.ShouldBindJSON(&req); err != nil {
			response.Validation(c, err.Error())
			return
		}
	}
	tasks, approval, err := h.agent.GeneratePlan(c.Request.Context(), user.ID, projectID, req.Replace)
	if err != nil {
		writeAgentError(c, err)
		return
	}
	c.JSON(http.StatusCreated, generatePlanResponse{Tasks: tasks, Approval: approval})
}

func (h *AgentHandler) TransitionTask(c *gin.Context) {
	user, ok := middleware.CurrentUser(c)
	if !ok {
		response.Error(c, http.StatusUnauthorized, "auth.unauthorized", "missing user")
		return
	}
	taskID, ok := parseUUIDParam(c, "id")
	if !ok {
		return
	}
	var req transitionTaskRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Validation(c, err.Error())
		return
	}
	task, err := h.agent.TransitionTask(c.Request.Context(), user.ID, taskID, req.Status, req.Note)
	if err != nil {
		writeAgentError(c, err)
		return
	}
	c.JSON(http.StatusOK, task)
}

func (h *AgentHandler) RequestTaskExecution(c *gin.Context) {
	user, ok := middleware.CurrentUser(c)
	if !ok {
		response.Error(c, http.StatusUnauthorized, "auth.unauthorized", "missing user")
		return
	}
	taskID, ok := parseUUIDParam(c, "id")
	if !ok {
		return
	}
	approval, err := h.agent.RequestTaskExecution(c.Request.Context(), user.ID, taskID)
	if err != nil {
		writeAgentError(c, err)
		return
	}
	c.JSON(http.StatusCreated, approval)
}

func (h *AgentHandler) ListApprovals(c *gin.Context) {
	user, ok := middleware.CurrentUser(c)
	if !ok {
		response.Error(c, http.StatusUnauthorized, "auth.unauthorized", "missing user")
		return
	}
	projectID, ok := parseUUIDParam(c, "id")
	if !ok {
		return
	}
	status := strings.TrimSpace(c.Query("status"))
	approvals, err := h.agent.ListApprovals(c.Request.Context(), user.ID, projectID, status)
	if err != nil {
		writeAgentError(c, err)
		return
	}
	c.JSON(http.StatusOK, approvalListResponse{Items: approvals})
}

func (h *AgentHandler) DecideApproval(c *gin.Context) {
	user, ok := middleware.CurrentUser(c)
	if !ok {
		response.Error(c, http.StatusUnauthorized, "auth.unauthorized", "missing user")
		return
	}
	approvalID, ok := parseUUIDParam(c, "id")
	if !ok {
		return
	}
	var req decideApprovalRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Validation(c, err.Error())
		return
	}
	approval, err := h.agent.DecideApproval(c.Request.Context(), user.ID, approvalID, req.Decision, req.Note)
	if err != nil {
		writeAgentError(c, err)
		return
	}
	c.JSON(http.StatusOK, approval)
}

func (h *AgentHandler) ListRuns(c *gin.Context) {
	user, ok := middleware.CurrentUser(c)
	if !ok {
		response.Error(c, http.StatusUnauthorized, "auth.unauthorized", "missing user")
		return
	}
	projectID, ok := parseUUIDParam(c, "id")
	if !ok {
		return
	}
	runs, err := h.agent.ListRuns(c.Request.Context(), user.ID, projectID)
	if err != nil {
		writeAgentError(c, err)
		return
	}
	c.JSON(http.StatusOK, runListResponse{Items: runs})
}

func (h *AgentHandler) FinishRun(c *gin.Context) {
	user, ok := middleware.CurrentUser(c)
	if !ok {
		response.Error(c, http.StatusUnauthorized, "auth.unauthorized", "missing user")
		return
	}
	runID, ok := parseUUIDParam(c, "id")
	if !ok {
		return
	}
	var req finishRunRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Validation(c, err.Error())
		return
	}
	run, err := h.agent.FinishRun(c.Request.Context(), user.ID, runID, req.Status, req.OutputSummary)
	if err != nil {
		writeAgentError(c, err)
		return
	}
	c.JSON(http.StatusOK, run)
}

func writeAgentError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, agentapp.ErrValidation):
		response.Validation(c, err.Error())
	case errors.Is(err, agentapp.ErrPlanExists):
		response.Error(c, http.StatusConflict, "agent.plan_exists", "task plan already exists")
	case errors.Is(err, domain.ErrNotFound):
		response.Error(c, http.StatusNotFound, "agent.not_found", "project, task, run or approval not found")
	default:
		response.Error(c, http.StatusInternalServerError, "internal_error", "agent operation failed")
	}
}
