package main

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"backend/internal/config"
	"backend/internal/domain"
	"backend/internal/infrastructure/migrate"
	"backend/internal/infrastructure/persistence"
	"backend/internal/infrastructure/postgres"
	"backend/internal/security"

	"github.com/google/uuid"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		exit("load config: %v", err)
	}
	if cfg.Database.RunMigrations {
		if err := migrate.Up(cfg.Database); err != nil {
			exit("run migrations: %v", err)
		}
	}

	db, err := postgres.Open(cfg.Database)
	if err != nil {
		exit("open postgres: %v", err)
	}
	defer func() { _ = db.Close() }()

	repos := persistence.NewRepositories(db.DB)
	ctx := context.Background()

	count, err := repos.Users.Count(ctx)
	if err != nil {
		exit("count users: %v", err)
	}
	if count > 0 {
		fmt.Println("seed skipped: a local user already exists")
		return
	}

	username := env("SEED_USERNAME", "admin")
	password := env("SEED_PASSWORD", "admin12345")
	displayName := env("SEED_DISPLAY_NAME", "Admin")
	username = strings.ToLower(strings.TrimSpace(username))
	if username == "" || password == "" {
		exit("SEED_USERNAME and SEED_PASSWORD must not be empty")
	}

	hasher := security.NewPasswordHasher()
	passwordHash, err := hasher.Hash(password)
	if err != nil {
		exit("hash password: %v", err)
	}
	id, err := uuid.NewV7()
	if err != nil {
		exit("generate user id: %v", err)
	}
	now := time.Now().UTC()
	user := &domain.User{
		ID:           id,
		Username:     username,
		PasswordHash: passwordHash,
		DisplayName:  displayName,
		Status:       domain.UserStatusActive,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	if err := repos.Users.Create(ctx, user); err != nil {
		exit("create seed user: %v", err)
	}

	fmt.Printf("seed user created\nusername: %s\npassword: %s\n", username, password)
}

func env(key, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(key)); value != "" {
		return value
	}
	return fallback
}

func exit(format string, args ...any) {
	_, _ = fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}
