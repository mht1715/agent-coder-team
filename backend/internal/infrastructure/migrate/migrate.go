package migrate

import (
	"errors"
	"fmt"
	"net/url"

	"backend/internal/config"
	"backend/migrations"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/golang-migrate/migrate/v4/source/iofs"
)

func Up(cfg config.DatabaseConfig) error {
	source, err := iofs.New(migrations.FS, ".")
	if err != nil {
		return fmt.Errorf("create migration source: %w", err)
	}

	databaseURL := &url.URL{
		Scheme: "postgres",
		User:   url.UserPassword(cfg.User, cfg.Password),
		Host:   fmt.Sprintf("%s:%d", cfg.Host, cfg.Port),
		Path:   cfg.Name,
	}
	query := databaseURL.Query()
	query.Set("sslmode", cfg.SSLMode)
	databaseURL.RawQuery = query.Encode()

	m, err := migrate.NewWithSourceInstance("iofs", source, databaseURL.String())
	if err != nil {
		_ = source.Close()
		return fmt.Errorf("create migrator: %w", err)
	}

	if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		_, _ = m.Close()
		return fmt.Errorf("run migrations: %w", err)
	}

	sourceErr, dbErr := m.Close()
	if sourceErr != nil {
		return fmt.Errorf("close migration source: %w", sourceErr)
	}
	if dbErr != nil {
		return fmt.Errorf("close migration database: %w", dbErr)
	}
	return nil
}
