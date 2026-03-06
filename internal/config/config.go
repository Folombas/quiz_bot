package config

import (
	"os"

	"gopkg.in/yaml.v3"
)

// Config — основная структура конфигурации
type Config struct {
	Bot     BotConfig     `yaml:"bot"`
	Database DatabaseConfig `yaml:"database"`
	Log     LogConfig     `yaml:"log"`
	RateLimit RateLimitConfig `yaml:"rate_limit"`
}

// BotConfig — настройки бота
type BotConfig struct {
	Token            string `yaml:"token"`
	TokenEnvVar      string `yaml:"token_env_var"`
	WebhookURL       string `yaml:"webhook_url"`
	MaxConnections   int    `yaml:"max_connections"`
	Timeout          int    `yaml:"timeout"`
}

// DatabaseConfig — настройки базы данных
type DatabaseConfig struct {
	Type string `yaml:"type"` // sqlite, postgres
	SQLite SQLiteConfig `yaml:"sqlite"`
	Postgres PostgresConfig `yaml:"postgres"`
}

// SQLiteConfig — настройки SQLite
type SQLiteConfig struct {
	Path string `yaml:"path"`
}

// PostgresConfig — настройки PostgreSQL
type PostgresConfig struct {
	Host     string `yaml:"host"`
	Port     int    `yaml:"port"`
	User     string `yaml:"user"`
	Password string `yaml:"password"`
	DBName   string `yaml:"db_name"`
	SSLMode  string `yaml:"ssl_mode"`
}

// LogConfig — настройки логирования
type LogConfig struct {
	Level  string `yaml:"level"`  // debug, info, warn, error
	Format string `yaml:"format"` // json, text
	Output string `yaml:"output"` // stdout, stderr, file
	File   string `yaml:"file"`
}

// RateLimitConfig — настройки ограничения запросов
type RateLimitConfig struct {
	Enabled        bool `yaml:"enabled"`
	RequestsPerMin int  `yaml:"requests_per_min"`
	BurstSize      int  `yaml:"burst_size"`
}

// Load загружает конфигурацию из файла
func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}

	// Загружаем токен из переменной окружения, если указано
	if cfg.Bot.TokenEnvVar != "" {
		cfg.Bot.Token = os.Getenv(cfg.Bot.TokenEnvVar)
	}

	return &cfg, nil
}

// DefaultConfig возвращает конфигурацию по умолчанию
func DefaultConfig() *Config {
	return &Config{
		Bot: BotConfig{
			TokenEnvVar:    "TELEGRAM_BOT_TOKEN",
			MaxConnections: 100,
			Timeout:        60,
		},
		Database: DatabaseConfig{
			Type: "sqlite",
			SQLite: SQLiteConfig{
				Path: "data/quiz_bot.db",
			},
		},
		Log: LogConfig{
			Level:  "info",
			Format: "text",
			Output: "stdout",
		},
		RateLimit: RateLimitConfig{
			Enabled:        true,
			RequestsPerMin: 30,
			BurstSize:      10,
		},
	}
}
