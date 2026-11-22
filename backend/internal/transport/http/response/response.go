package response

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

type ErrorBody struct {
	Code      string `json:"code"`
	Message   string `json:"message"`
	RequestID string `json:"request_id,omitempty"`
	Details   any    `json:"details,omitempty"`
}

func Error(c *gin.Context, status int, code, message string) {
	c.AbortWithStatusJSON(status, ErrorBody{
		Code:      code,
		Message:   message,
		RequestID: c.GetString("request_id"),
	})
}

func Validation(c *gin.Context, message string) {
	Error(c, http.StatusBadRequest, "validation.invalid_request", message)
}
