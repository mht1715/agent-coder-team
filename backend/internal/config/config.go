package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
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
	Driver                 string   `mapstructure:"driver"`
	Host                   string   `mapstructure:"host"`
	Network                string   `mapstructure:"network"`
	CPUs                   float64  `mapstructure:"cpus"`
	MemoryMB               int      `mapstructure:"memory_mb"`
	PidsLimit              int      `mapstructure:"pids_limit"`
	TimeoutSeconds         int      `mapstructure:"timeout_seconds"`
	WorkspaceAnchor        string   `mapstructure:"workspace_anchor"`
	ContainerWorkspaceRoot string   `mapstructure:"container_workspace_root"`
	MaxConcurrent          int      `mapstructure:"max_concurrent"`
	MaxOutputBytes         int      `mapstructure:"max_output_bytes"`
	DefaultImage           string   `mapstructure:"default_image"`
	Images                 []string `mapstructure:"images"`
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
	v.SetEnvPrefix("ACT")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()
	v.SetDefault("server.port", 18080)
	v.SetDefault("server.host", "127.0.0.1")
	v.SetDefault("sandbox.driver", "docker")
	v.SetDefault("sandbox.host", "")
	v.SetDefault("sandbox.workspace_anchor", "")
	v.SetDefault("sandbox.container_workspace_root", "")
	v.SetDefault("sandbox.network", "none")
	v.SetDefault("sandbox.cpus", 2)
	v.SetDefault("sandbox.memory_mb", 2048)
	v.SetDefault("sandbox.pids_limit", 256)
	v.SetDefault("sandbox.timeout_seconds", 1800)
	v.SetDefault("sandbox.max_concurrent", 1)
	v.SetDefault("sandbox.max_output_bytes", 200000)
	v.SetDefault("sandbox.default_image", "alpine:3.20")
	v.SetDefault("sandbox.images", []string{
		"alpine:3.20",
		"golang:1.27-alpine",
		"node:22-alpine",
		"python:3.13-alpine",
		"eclipse-temurin:21-jdk-alpine",
		"mcr.microsoft.com/dotnet/sdk:8.0-alpine",
	})

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
	if strings.EqualFold(c.Sandbox.Driver, "docker") {
		if strings.TrimSpace(c.Sandbox.DefaultImage) == "" {
			return fmt.Errorf("sandbox.default_image is required when sandbox.driver is docker")
		}
		if len(c.Sandbox.Images) == 0 {
			return fmt.Errorf("sandbox.images must contain at least one allowed image")
		}
		if c.Sandbox.TimeoutSeconds <= 0 {
			c.Sandbox.TimeoutSeconds = 1800
		}
		if c.Sandbox.MaxConcurrent <= 0 {
			c.Sandbox.MaxConcurrent = 1
		}
	}
	return nil
}
