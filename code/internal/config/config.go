package config

import (
	"log/slog"
	"os"
	"strconv"
	"time"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Server   ServerConfig   `yaml:"server"`
	Database DatabaseConfig `yaml:"database"`
	Redis    RedisConfig    `yaml:"redis"`
	JWT      JWTConfig      `yaml:"jwt"`
}

type JWTConfig struct {
	Secret string `yaml:"secret"`
}

type ServerConfig struct {
	Addr string `yaml:"addr"`
}

type DatabaseConfig struct {
	URL             string        `yaml:"url"`
	MaxOpenConns    int           `yaml:"max_open_conns"`
	MaxIdleConns    int           `yaml:"max_idle_conns"`
	ConnMaxLifetime time.Duration `yaml:"conn_max_lifetime"`
}

type RedisConfig struct {
	Addr     string `yaml:"addr"`
	Password string `yaml:"password"`
	DB       int    `yaml:"db"`
}

func Load() *Config {
	cfg := &Config{
		Server:   ServerConfig{Addr: ":8080"},
		Database: DatabaseConfig{MaxOpenConns: 25, MaxIdleConns: 5, ConnMaxLifetime: 5 * time.Minute},
	}

	profile := os.Getenv("TICK_PROFILE")
	if profile == "" {
		profile = "sit"
	}
	paths := []string{"configs/application.yml", "configs/application-" + profile + ".yml"}

	for _, p := range paths {
		data, err := os.ReadFile(p)
		if err != nil {
			continue
		}
		yaml.Unmarshal(data, cfg)
	}

	// env vars override config file
	if v := os.Getenv("TICK_SERVER_ADDR"); v != "" {
		cfg.Server.Addr = v
	}
	if v := os.Getenv("TICK_DATABASE_URL"); v != "" {
		cfg.Database.URL = v
	}
	if v := os.Getenv("TICK_REDIS_ADDR"); v != "" {
		cfg.Redis.Addr = v
	}
	if v := os.Getenv("TICK_REDIS_PASSWORD"); v != "" {
		cfg.Redis.Password = v
	}
	if v := os.Getenv("TICK_REDIS_DB"); v != "" {
		if i, err := strconv.Atoi(v); err == nil {
			cfg.Redis.DB = i
		}
	}
	if v := os.Getenv("JWT_SECRET"); v != "" {
		cfg.JWT.Secret = v
	}

	return cfg
}

func InitLogger() {
	handler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})
	slog.SetDefault(slog.New(handler))
}
