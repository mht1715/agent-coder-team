package config

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/viper"
)

type SysConfig struct {
	App       AppConfig       `mapstructure:"app"`
	Server    ServerConfig    `mapstructure:"server"`
	Database  DatabaseConfig  `mapstructure:"database"`
	Redis     RedisConfig     `mapstructure:"redis"`
	Queue     QueueConfig     `mapstructure:"queue"`
	Vector    VectorConfig    `mapstructure:"vector"`
	Sandbox   SandboxConfig   `mapstructure:"sandbox"`
	Auth      AuthConfig      `mapstructure:"auth"`
	Security  SecurityConfig  `mapstructure:"security"`
	Workspace WorkspaceConfig `mapstructure:"workspace"`
	Scheduler SchedulerConfig `mapstructure:"scheduler"`
	I18n      I18nConfig      `mapstructure:"i18n"`
}

type AppConfig struct {
	Name        string `mapstructure:"name"`
	Version     string `mapstructure:"version"`
	Environment string `mapstructure:"environment"`
	GinMode     string `mapstructure:"gin_mode"`
}

type ServerConfig struct {
	Host            string `mapstructure:"host"`
	Port            int    `mapstructure:"port"`
	ReadTimeout     string `mapstructure:"read_timeout"`
	WriteTimeout    string `mapstructure:"write_timeout"`
	ShutdownTimeout string `mapstructure:"shutdown_timeout"`
}

type DatabaseConfig struct {
	Host            string `mapstructure:"host"`
	Port            int    `mapstructure:"port"`
	User            string `mapstructure:"user"`
	Password        string `mapstructure:"password"`
	Name            string `mapstructure:"name"`
	SSLMode         string `mapstructure:"sslmode"`
	MaxOpenConns    int    `mapstructure:"max_open_conns"`
	MaxIdleConns    int    `mapstructure:"max_idle_conns"`
	ConnMaxLifetime string `mapstructure:"conn_max_lifetime"`
	RunMigrations   bool   `mapstructure:"run_migrations"`
}

type RedisConfig struct {
	Host     string `mapstructure:"host"`
	Port     int    `mapstructure:"port"`
	Password string `mapstructure:"password"`
	DB       int    `mapstructure:"db"`
	PoolSize int    `mapstructure:"pool_size"`
}

type QueueConfig struct {
	Driver      string `mapstructure:"driver"`
	Concurrency int    `mapstructure:"concurrency"`
	RedisDB     int    `mapstructure:"redis_db"`
}

type VectorConfig struct {
	Driver    string `mapstructure:"driver"`
	Dimension int    `mapstructure:"dimension"`
}

type SandboxConfig struct {
	Driver         string  `mapstructure:"driver"`
	Host           string  `mapstructure:"host"`
	Network        string  `mapstructure:"network"`
	CPUs           float64 `mapstructure:"cpus"`
	MemoryMB       int     `mapstructure:"memory_mb"`
	PidsLimit      int     `mapstructure:"pids_limit"`
	TimeoutSeconds int     `mapstructure:"timeout_seconds"`
}

type AuthConfig struct {
	SessionTTL string `mapstructure:"session_ttl"`
	RefreshTTL string `mapstructure:"refresh_ttl"`
}

type SecurityConfig struct {
	ForceHTTPS     bool     `mapstructure:"force_https"`
	AllowedOrigins []string `mapstructure:"allowed_origins"`
}

type WorkspaceConfig struct {
	AllowedRoots []string `mapstructure:"allowed_roots"`
	MaxScanFiles int      `mapstructure:"max_scan_files"`
}

type SchedulerConfig struct {
	Timezone            string   `mapstructure:"timezone"`
	AutoStartWeekdays   []string `mapstructure:"auto_start_weekdays"`
	AutoStartHolidays   []string `mapstructure:"auto_start_holidays"`
	AllowManualOverride bool     `mapstructure:"allow_manual_override"`
}

type I18nConfig struct {
	DefaultLocale    string   `mapstructure:"default_locale"`
	SupportedLocales []string `mapstructure:"supported_locales"`
}

func Load() (*SysConfig, error) {
	configPath, err := resolveConfigPath()
	if err != nil {
		return nil, err
	}

	v := viper.New()
	v.SetConfigFile(configPath)
	v.SetDefault("server.port", 18080)
	v.SetDefault("server.host", "127.0.0.1")

	if err := v.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("read config %s: %w", configPath, err)
	}

	var cfg SysConfig
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("unmarshal config: %w", err)
	}
	return &cfg, nil
}

func resolveConfigPath() (string, error) {
	if path := os.Getenv("CONFIG_PATH"); path != "" {
		if _, err := os.Stat(path); err != nil {
			return "", fmt.Errorf("config file not found: %w", err)
		}
		return path, nil
	}

	wd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("get working directory: %w", err)
	}

	candidates := []string{
		filepath.Join(wd, "config", "config.yml"),
		filepath.Join(wd, "config", "config.yaml"),
		filepath.Join(wd, "backend", "config", "config.yml"),
		filepath.Join(wd, "backend", "config", "config.yaml"),
	}
	for _, candidate := range candidates {
		if _, err := os.Stat(candidate); err == nil {
			return candidate, nil
		}
	}
	return "", fmt.Errorf("config file not found; set CONFIG_PATH or create config/config.yml")
}

func (c *SysConfig) Validate() error {
	if c.App.Name == "" {
		return fmt.Errorf("app.name is required")
	}
	switch c.App.GinMode {
	case "debug", "release", "test":
	default:
		return fmt.Errorf("app.gin_mode must be debug, release or test")
	}
	if c.Server.Host == "" {
		return fmt.Errorf("server.host is required")
	}
	if c.Server.Port < 1 || c.Server.Port > 65535 {
		return fmt.Errorf("server.port must be between 1 and 65535")
	}
	if c.Database.Host == "" || c.Database.User == "" || c.Database.Name == "" {
		return fmt.Errorf("database.host, database.user and database.name are required")
	}
	if c.Database.Port < 1 || c.Database.Port > 65535 {
		return fmt.Errorf("database.port must be between 1 and 65535")
	}
	if c.Redis.Host == "" {
		return fmt.Errorf("redis.host is required")
	}
	if c.Redis.Port < 1 || c.Redis.Port > 65535 {
		return fmt.Errorf("redis.port must be between 1 and 65535")
	}
	if len(c.Workspace.AllowedRoots) == 0 {
		return fmt.Errorf("workspace.allowed_roots must contain at least one path")
	}
	if c.Workspace.MaxScanFiles <= 0 {
		c.Workspace.MaxScanFiles = 20000
	}
	if c.Vector.Dimension <= 0 {
		return fmt.Errorf("vector.dimension must be positive")
	}
	if c.Scheduler.Timezone != "" {
		if _, err := time.LoadLocation(c.Scheduler.Timezone); err != nil {
			return fmt.Errorf("scheduler.timezone is invalid: %w", err)
		}
	}
	return nil
}
