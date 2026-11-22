package main

import (
	"context"
	_ "embed"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"backend/internal/app"
	"backend/internal/config"
	"backend/internal/logger"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"
)

//go:embed banner.txt
var banner string

func main() {
	cfg, err := config.Load()
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "load config failed: %v\n", err)
		os.Exit(1)
	}

	logger.Init(cfg.App)

	switch cfg.App.GinMode {
	case "release":
		gin.SetMode(gin.ReleaseMode)
	case "test":
		gin.SetMode(gin.TestMode)
	default:
		gin.SetMode(gin.DebugMode)
	}

	if banner != "" {
		fmt.Println(banner)
	}

	application, err := app.New(cfg)
	if err != nil {
		log.Error().Err(err).Msg("initialize application failed")
		os.Exit(1)
	}
	defer func() {
		if err := application.Close(); err != nil {
			log.Error().Err(err).Msg("close application failed")
		}
	}()

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	if err := application.Run(ctx); err != nil {
		log.Error().Err(err).Msg("application stopped with error")
		os.Exit(1)
	}
}
