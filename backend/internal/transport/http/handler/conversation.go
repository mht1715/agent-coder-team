package handler

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	conversationapp "backend/internal/application/conversation"
	"backend/internal/domain"
	"backend/internal/transport/http/middleware"
	"backend/internal/transport/http/response"

	"github.com/gin-gonic/gin"
)

type ConversationHandler struct {
	conversations *conversationapp.Service
}

type conversationsResponse struct {
	Items []domain.Conversation `json:"items"`
}

type messagesResponse struct {
	Items      []domain.Message `json:"items"`
	HasMore    bool             `json:"has_more"`
	NextBefore *time.Time       `json:"next_before,omitempty"`
}

type sendMessageRequest struct {
	Content       string `json:"content" binding:"required"`
	ContentFormat string `json:"content_format"`
}

func NewConversationHandler(conversations *conversationapp.Service) *ConversationHandler {
	return &ConversationHandler{conversations: conversations}
}

func (h *ConversationHandler) ListProjectConversations(c *gin.Context) {
	user, ok := middleware.CurrentUser(c)
	if !ok {
		response.Error(c, http.StatusUnauthorized, "auth.unauthorized", "missing user")
		return
	}
	projectID, ok := parseUUIDParam(c, "id")
	if !ok {
		return
	}

	conversations, err := h.conversations.ListProjectConversations(c.Request.Context(), user.ID, projectID)
	if err != nil {
		writeConversationError(c, err)
		return
	}
	c.JSON(http.StatusOK, conversationsResponse{Items: conversations})
}

func (h *ConversationHandler) ListMessages(c *gin.Context) {
	user, ok := middleware.CurrentUser(c)
	if !ok {
		response.Error(c, http.StatusUnauthorized, "auth.unauthorized", "missing user")
		return
	}
	conversationID, ok := parseUUIDParam(c, "id")
	if !ok {
		return
	}

	var before *time.Time
	if raw := c.Query("before"); raw != "" {
		parsed, err := time.Parse(time.RFC3339Nano, raw)
		if err != nil {
			response.Validation(c, "before must be RFC3339 timestamp")
			return
		}
		before = &parsed
	}

	limit := 50
	if raw := c.Query("limit"); raw != "" {
		parsed, err := parseInt(raw)
		if err != nil || parsed < 1 || parsed > 100 {
			response.Validation(c, "limit must be between 1 and 100")
			return
		}
		limit = parsed
	}

	messages, hasMore, err := h.conversations.ListMessages(
		c.Request.Context(), user.ID, conversationID, before, limit,
	)
	if err != nil {
		writeConversationError(c, err)
		return
	}

	var nextBefore *time.Time
	if hasMore && len(messages) > 0 {
		last := messages[len(messages)-1].CreatedAt
		nextBefore = &last
	}
	c.JSON(http.StatusOK, messagesResponse{
		Items:      messages,
		HasMore:    hasMore,
		NextBefore: nextBefore,
	})
}

func (h *ConversationHandler) SendMessage(c *gin.Context) {
	user, ok := middleware.CurrentUser(c)
	if !ok {
		response.Error(c, http.StatusUnauthorized, "auth.unauthorized", "missing user")
		return
	}
	conversationID, ok := parseUUIDParam(c, "id")
	if !ok {
		return
	}

	var req sendMessageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Validation(c, err.Error())
		return
	}

	message, err := h.conversations.SendMessage(
		c.Request.Context(), user.ID, conversationID, req.Content, req.ContentFormat,
	)
	if err != nil {
		writeConversationError(c, err)
		return
	}
	c.JSON(http.StatusCreated, message)
}

func (h *ConversationHandler) Stream(c *gin.Context) {
	user, ok := middleware.CurrentUser(c)
	if !ok {
		response.Error(c, http.StatusUnauthorized, "auth.unauthorized", "missing user")
		return
	}
	conversationID, ok := parseUUIDParam(c, "id")
	if !ok {
		return
	}
	if _, err := h.conversations.GetConversation(c.Request.Context(), user.ID, conversationID); err != nil {
		writeConversationError(c, err)
		return
	}

	events, unsubscribe := h.conversations.Subscribe(conversationID)
	defer unsubscribe()

	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache, no-transform")
	c.Header("Connection", "keep-alive")
	c.Header("X-Accel-Buffering", "no")
	c.Status(http.StatusOK)

	// SSE is a long-lived response; clear the server-wide write deadline for this stream.
	_ = http.NewResponseController(c.Writer).SetWriteDeadline(time.Time{})

	if _, err := fmt.Fprint(c.Writer, "retry: 3000\n\n: connected\n\n"); err != nil {
		return
	}
	c.Writer.Flush()

	heartbeat := time.NewTicker(15 * time.Second)
	defer heartbeat.Stop()

	for {
		select {
		case <-c.Request.Context().Done():
			return
		case event, ok := <-events:
			if !ok {
				return
			}
			payload, err := json.Marshal(event.Data)
			if err != nil {
				continue
			}
			if _, err := fmt.Fprintf(
				c.Writer,
				"id: %s\nevent: %s\ndata: %s\n\n",
				event.ID,
				event.Type,
				payload,
			); err != nil {
				return
			}
			c.Writer.Flush()
		case <-heartbeat.C:
			if _, err := fmt.Fprint(c.Writer, ": ping\n\n"); err != nil {
				return
			}
			c.Writer.Flush()
		}
	}
}

func writeConversationError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, conversationapp.ErrValidation):
		response.Validation(c, err.Error())
	case errors.Is(err, domain.ErrNotFound):
		response.Error(c, http.StatusNotFound, "conversation.not_found", "conversation or project not found")
	default:
		response.Error(c, http.StatusInternalServerError, "internal_error", "conversation operation failed")
	}
}

func parseInt(value string) (int, error) {
	var parsed int
	_, err := fmt.Sscanf(value, "%d", &parsed)
	return parsed, err
}
