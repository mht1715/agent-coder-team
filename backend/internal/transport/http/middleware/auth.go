package middleware

import (
	"strings"

	"backend/internal/application/auth"
	"backend/internal/domain"
	"backend/internal/logger"
	"backend/internal/transport/http/response"

	"github.com/gin-gonic/gin"
)

const (
	CurrentUserKey    = "auth.user"
	CurrentSessionKey = "auth.session"
)

func RequireAuth(authSvc *auth.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		header := strings.TrimSpace(c.GetHeader("Authorization"))
		if !strings.HasPrefix(header, "Bearer ") {
			response.Error(c, 401, "auth.unauthorized", "missing bearer token")
			return
		}

		token := strings.TrimSpace(strings.TrimPrefix(header, "Bearer "))
		user, session, err := authSvc.Authenticate(c.Request.Context(), token)
		if err != nil {
			response.Error(c, 401, "auth.unauthorized", "invalid or expired token")
			return
		}

		c.Set(CurrentUserKey, user)
		c.Set(CurrentSessionKey, session)
		c.Request = c.Request.WithContext(logger.WithUserID(c.Request.Context(), user.ID.String()))
		c.Next()
	}
}

func CurrentUser(c *gin.Context) (*domain.User, bool) {
	value, ok := c.Get(CurrentUserKey)
	if !ok {
		return nil, false
	}
	user, ok := value.(*domain.User)
	return user, ok
}

func CurrentSession(c *gin.Context) (*domain.Session, bool) {
	value, ok := c.Get(CurrentSessionKey)
	if !ok {
		return nil, false
	}
	session, ok := value.(*domain.Session)
	return session, ok
}
