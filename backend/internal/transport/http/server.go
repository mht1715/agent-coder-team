package http

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"backend/internal/config"
	"backend/internal/health"
	"backend/internal/transport/http/handler"
	"backend/internal/transport/http/middleware"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"
)

type Server struct {
	cfg        *config.SysConfig
	httpServer *http.Server
}

func New(cfg *config.SysConfig, healthSvc *health.Service) *Server {
	h := handler.NewHealthHandler(healthSvc, cfg.App.Name, cfg.App.Version, cfg.App.Environment)

	router := gin.New()
	router.Use(
		gin.Recovery(),
		middleware.RequestID(),
		middleware.AccessLog(),
	)

	router.GET("/healthz", h.Healthz)
	router.GET("/readyz", h.Readyz)

	v1 := router.Group("/api/v1")
	v1.GET("/meta", h.Meta)

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
