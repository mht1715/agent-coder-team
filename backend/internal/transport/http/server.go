package http

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	agentapp "backend/internal/application/agent"
	authapp "backend/internal/application/auth"
	conversationapp "backend/internal/application/conversation"
	profileapp "backend/internal/application/profile"
	projectapp "backend/internal/application/project"
	workspaceapp "backend/internal/application/workspace"
	"backend/internal/config"
	"backend/internal/health"
	"backend/internal/transport/http/handler"
	"backend/internal/transport/http/middleware"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"
)

type Server struct {
	cfg        *config.SysConfig
	httpServer *http.Server
}

func New(
	cfg *config.SysConfig,
	healthSvc *health.Service,
	authSvc *authapp.Service,
	projectSvc *projectapp.Service,
	conversationSvc *conversationapp.Service,
	profileSvc *profileapp.Service,
	workspaceSvc *workspaceapp.Service,
	agentSvc *agentapp.Service,
) *Server {
	healthHandler := handler.NewHealthHandler(healthSvc, cfg.App.Name, cfg.App.Version, cfg.App.Environment)
	authHandler := handler.NewAuthHandler(authSvc)
	projectHandler := handler.NewProjectHandler(projectSvc)
	conversationHandler := handler.NewConversationHandler(conversationSvc, agentSvc)
	profileHandler := handler.NewProfileHandler(profileSvc)
	workspaceHandler := handler.NewWorkspaceHandler(workspaceSvc)
	agentHandler := handler.NewAgentHandler(agentSvc)

	router := gin.New()
	corsConfig := cors.Config{
		AllowMethods:     []string{"GET", "POST", "PATCH", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization", "X-Request-ID"},
		ExposeHeaders:    []string{"X-Request-ID"},
		AllowCredentials: false,
		MaxAge:           12 * time.Hour,
	}
	if cfg.App.Environment == "development" {
		corsConfig.AllowAllOrigins = true
	} else {
		corsConfig.AllowOrigins = cfg.Security.AllowedOrigins
	}
	router.Use(
		cors.New(corsConfig),
		gin.Recovery(),
		middleware.RequestID(),
		middleware.AccessLog(),
	)

	router.GET("/healthz", healthHandler.Healthz)
	router.GET("/readyz", healthHandler.Readyz)

	v1 := router.Group("/api/v1")
	v1.GET("/meta", healthHandler.Meta)
	v1.GET("/auth/bootstrap", authHandler.BootstrapStatus)
	v1.POST("/auth/register", authHandler.Register)
	v1.POST("/auth/login", authHandler.Login)
	v1.POST("/auth/refresh", authHandler.Refresh)

	protected := v1.Group("")
	protected.Use(middleware.RequireAuth(authSvc))
	protected.POST("/auth/logout", authHandler.Logout)
	protected.GET("/me", authHandler.Me)
	protected.POST("/projects", projectHandler.Create)
	protected.GET("/projects", projectHandler.List)
	protected.GET("/projects/:id", projectHandler.Get)
	protected.PATCH("/projects/:id", projectHandler.Update)
	protected.GET("/workspaces/browse", workspaceHandler.Browse)
	protected.GET("/projects/:id/profile", profileHandler.Get)
	protected.POST("/projects/:id/profile/scan", profileHandler.Scan)
	protected.GET("/projects/:id/conversations", conversationHandler.ListProjectConversations)
	protected.GET("/conversations/:id/messages", conversationHandler.ListMessages)
	protected.POST("/conversations/:id/messages", conversationHandler.SendMessage)
	protected.GET("/conversations/:id/stream", conversationHandler.Stream)

	protected.GET("/projects/:id/requirement", agentHandler.GetRequirement)
	protected.POST("/projects/:id/requirement/generate", agentHandler.GenerateRequirement)
	protected.PATCH("/projects/:id/requirement", agentHandler.UpdateRequirement)
	protected.POST("/projects/:id/requirement/approve", agentHandler.ApproveRequirement)
	protected.GET("/projects/:id/tasks", agentHandler.ListTasks)
	protected.POST("/projects/:id/tasks/generate", agentHandler.GeneratePlan)
	protected.POST("/tasks/:id/transition", agentHandler.TransitionTask)
	protected.POST("/tasks/:id/request-execution", agentHandler.RequestTaskExecution)
	protected.GET("/projects/:id/approvals", agentHandler.ListApprovals)
	protected.POST("/approvals/:id/decision", agentHandler.DecideApproval)
	protected.GET("/projects/:id/runs", agentHandler.ListRuns)
	protected.POST("/runs/:id/finish", agentHandler.FinishRun)

	addr := fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port)
	return &Server{
		cfg: cfg,
		httpServer: &http.Server{
			Addr:              addr,
			Handler:           router,
			ReadHeaderTimeout: duration(cfg.Server.ReadTimeout, 5*time.Second),
			ReadTimeout:       duration(cfg.Server.ReadTimeout, 15*time.Second),
			WriteTimeout:      duration(cfg.Server.WriteTimeout, 60*time.Second),
			IdleTimeout:       60 * time.Second,
		},
	}
}

func (s *Server) Run(ctx context.Context) error {
	errCh := make(chan error, 1)

	go func() {
		log.Info().
			Str("addr", s.httpServer.Addr).
			Str("gin_mode", gin.Mode()).
			Msg("HTTP server started")

		if err := s.httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
		close(errCh)
	}()

	select {
	case err := <-errCh:
		if err != nil {
			return fmt.Errorf("http server: %w", err)
		}
		return nil
	case <-ctx.Done():
		log.Info().Msg("shutdown signal received")
	}

	shutdownCtx, cancel := context.WithTimeout(
		context.Background(),
		duration(s.cfg.Server.ShutdownTimeout, 10*time.Second),
	)
	defer cancel()

	if err := s.httpServer.Shutdown(shutdownCtx); err != nil {
		_ = s.httpServer.Close()
		return fmt.Errorf("graceful shutdown: %w", err)
	}

	log.Info().Msg("HTTP server stopped")
	return nil
}

func duration(value string, fallback time.Duration) time.Duration {
	if value == "" {
		return fallback
	}
	parsed, err := time.ParseDuration(value)
	if err != nil || parsed <= 0 {
		return fallback
	}
	return parsed
}
