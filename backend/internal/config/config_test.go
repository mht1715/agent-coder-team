package config

import "testing"

func validConfig() SysConfig {
	return SysConfig{
		App: AppConfig{
			Name:    "agent-coder-team",
			GinMode: "debug",
		},
		Server: ServerConfig{
			Host: "127.0.0.1",
			Port: 18080,
		},
		Database: DatabaseConfig{
			Host: "postgres",
			Port: 5432,
			User: "act",
			Name: "agent_coder_team",
		},
		Redis: RedisConfig{
			Host: "redis",
			Port: 6379,
		},
		Vector: VectorConfig{
			Dimension: 1536,
		},
		Scheduler: SchedulerConfig{
			Timezone: "Asia/Shanghai",
		},
	}
}

func TestValidate(t *testing.T) {
	t.Run("valid config", func(t *testing.T) {
		cfg := validConfig()
		if err := cfg.Validate(); err != nil {
			t.Fatalf("expected valid config, got %v", err)
		}
	})

	t.Run("invalid server port", func(t *testing.T) {
		cfg := validConfig()
		cfg.Server.Port = 0
		if err := cfg.Validate(); err == nil {
			t.Fatal("expected invalid server port error")
		}
	})

	t.Run("invalid timezone", func(t *testing.T) {
		cfg := validConfig()
		cfg.Scheduler.Timezone = "Invalid/Timezone"
		if err := cfg.Validate(); err == nil {
			t.Fatal("expected invalid timezone error")
		}
	})
}
