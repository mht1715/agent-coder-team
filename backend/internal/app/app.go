package app

import (
	"context"
	"fmt"

	"backend/internal/config"
	"backend/internal/health"
	"backend/internal/infrastructure/cache"
	"backend/internal/infrastructure/postgres"
	httpserver "backend/internal/transport/http"

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

	redisClient, err := cache.Open(cfg.Redis)
	if err != nil {
		_ = db.Close()
		return nil, err
	}

	healthSvc := health.New(db, redisClient)
	server := httpserver.New(cfg, healthSvc)

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
