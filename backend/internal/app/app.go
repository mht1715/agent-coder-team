package app

import (
	"context"
	"fmt"
	"time"

	"backend/internal/analyzer"
	agentapp "backend/internal/application/agent"
	"backend/internal/application/auth"
	"backend/internal/application/conversation"
	"backend/internal/application/profile"
	"backend/internal/application/project"
	workspaceapp "backend/internal/application/workspace"
	"backend/internal/config"
	"backend/internal/event"
	"backend/internal/health"
	"backend/internal/infrastructure/cache"
	"backend/internal/infrastructure/migrate"
	"backend/internal/infrastructure/persistence"
	"backend/internal/infrastructure/postgres"
	"backend/internal/security"
	httpserver "backend/internal/transport/http"
	"backend/internal/workspace"

	"github.com/rs/zerolog/log"
)

type App struct {
	cfg    *config.SysConfig
	db     *postgres.Client
	cache  *cache.Client
	server *httpserver.Server
}

func New(cfg *config.SysConfig) (*App, error) {
	db, err := postgres.Open(cfg.Database)
	if err != nil {
		return nil, err
	}

	if cfg.Database.RunMigrations {
		if err := migrate.Up(cfg.Database); err != nil {
			_ = db.Close()
			return nil, err
		}
	}

	redisClient, err := cache.Open(cfg.Redis)
	if err != nil {
		_ = db.Close()
		return nil, err
	}

	accessTTL, err := time.ParseDuration(cfg.Auth.SessionTTL)
	if err != nil {
		_ = redisClient.Close()
		_ = db.Close()
		return nil, fmt.Errorf("parse auth.session_ttl: %w", err)
	}
	refreshTTL, err := time.ParseDuration(cfg.Auth.RefreshTTL)
	if err != nil {
		_ = redisClient.Close()
		_ = db.Close()
		return nil, fmt.Errorf("parse auth.refresh_ttl: %w", err)
	}

	repositories := persistence.NewRepositories(db.DB)
	authService := auth.NewService(
		repositories.Users,
		repositories.Sessions,
		security.NewPasswordHasher(),
		accessTTL,
		refreshTTL,
	)
	workspaceValidator, err := workspace.NewValidator(cfg.Workspace)
	if err != nil {
		_ = redisClient.Close()
		_ = db.Close()
		return nil, err
	}
	projectService := project.NewService(repositories.Projects, workspaceValidator)
	workspaceService := workspaceapp.NewService(workspaceValidator)
	profileService := profile.NewService(
		repositories.Projects,
		repositories.Profiles,
		workspaceValidator,
		analyzer.NewDetector(workspaceValidator.MaxScanFiles()),
	)
	eventBroker := event.NewBroker()
	conversationService := conversation.NewService(
		repositories.Projects,
		repositories.Conversations,
		repositories.Messages,
		eventBroker,
	)
	agentService := agentapp.NewService(
		repositories.Projects,
		repositories.Profiles,
		repositories.Conversations,
		repositories.Messages,
		repositories.Requirements,
		repositories.Tasks,
		repositories.Approvals,
		repositories.Runs,
		eventBroker,
	)
	healthSvc := health.New(db, redisClient)
	server := httpserver.New(cfg, healthSvc, authService, projectService, conversationService, profileService, workspaceService, agentService)

	return &App{
		cfg:    cfg,
		db:     db,
		cache:  redisClient,
		server: server,
	}, nil
}

func (a *App) Run(ctx context.Context) error {
	log.Info().
		Str("app", a.cfg.App.Name).
		Str("version", a.cfg.App.Version).
		Str("environment", a.cfg.App.Environment).
		Msg("application starting")

	return a.server.Run(ctx)
}

func (a *App) Close() error {
	var closeErr error
	if a.cache != nil {
		if err := a.cache.Close(); err != nil {
			closeErr = fmt.Errorf("close redis: %w", err)
		}
	}
	if a.db != nil {
		if err := a.db.Close(); err != nil && closeErr == nil {
			closeErr = fmt.Errorf("close postgres: %w", err)
		}
	}
	return closeErr
}
