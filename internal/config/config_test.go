package config

import (
	"os"
	"testing"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	if cfg.Bot.TokenEnvVar != "TELEGRAM_BOT_TOKEN" {
		t.Errorf("Expected TokenEnvVar to be TELEGRAM_BOT_TOKEN, got %s", cfg.Bot.TokenEnvVar)
	}

	if cfg.Bot.MaxConnections != 100 {
		t.Errorf("Expected MaxConnections to be 100, got %d", cfg.Bot.MaxConnections)
	}

	if cfg.Database.Type != "sqlite" {
		t.Errorf("Expected Database.Type to be sqlite, got %s", cfg.Database.Type)
	}

	if cfg.Log.Level != "info" {
		t.Errorf("Expected Log.Level to be info, got %s", cfg.Log.Level)
	}

	if !cfg.RateLimit.Enabled {
		t.Error("Expected RateLimit.Enabled to be true")
	}
}

func TestLoad(t *testing.T) {
	// Создаём временный файл конфигурации
	tmpFile := "/tmp/test_config.yaml"
	content := `
bot:
  token_env_var: TEST_TOKEN
  max_connections: 50
  timeout: 30

database:
  type: sqlite
  sqlite:
    path: test.db

log:
  level: debug
  format: json
  output: stdout

rate_limit:
  enabled: false
  requests_per_min: 10
  burst_size: 5
`
	if err := os.WriteFile(tmpFile, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile)

	// Устанавливаем переменную окружения
	os.Setenv("TEST_TOKEN", "test_bot_token")
	defer os.Unsetenv("TEST_TOKEN")

	// Загружаем конфигурацию
	cfg, err := Load(tmpFile)
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	// Проверяем значения
	if cfg.Bot.Token != "test_bot_token" {
		t.Errorf("Expected Token to be test_bot_token, got %s", cfg.Bot.Token)
	}

	if cfg.Bot.MaxConnections != 50 {
		t.Errorf("Expected MaxConnections to be 50, got %d", cfg.Bot.MaxConnections)
	}

	if cfg.Log.Level != "debug" {
		t.Errorf("Expected Log.Level to be debug, got %s", cfg.Log.Level)
	}

	if cfg.Log.Format != "json" {
		t.Errorf("Expected Log.Format to be json, got %s", cfg.Log.Format)
	}

	if cfg.RateLimit.Enabled {
		t.Error("Expected RateLimit.Enabled to be false")
	}
}

func TestLoad_NonExistentFile(t *testing.T) {
	_, err := Load("/nonexistent/path/config.yaml")
	if err == nil {
		t.Error("Expected error for non-existent file")
	}
}

func TestLoad_InvalidYAML(t *testing.T) {
	tmpFile := "/tmp/invalid_config.yaml"
	content := `invalid: yaml: content: [`
	if err := os.WriteFile(tmpFile, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile)

	_, err := Load(tmpFile)
	if err == nil {
		t.Error("Expected error for invalid YAML")
	}
}
