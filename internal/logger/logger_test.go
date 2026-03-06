package logger

import (
	"testing"

	"quiz_bot/internal/config"
)

func TestNew(t *testing.T) {
	cfg := config.LogConfig{
		Level:  "info",
		Format: "text",
		Output: "stdout",
	}

	l, err := New(cfg)
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}

	if l == nil {
		t.Fatal("Expected logger to be created")
	}
}

func TestNew_InvalidLevel(t *testing.T) {
	cfg := config.LogConfig{
		Level:  "invalid",
		Format: "text",
		Output: "stdout",
	}

	l, err := New(cfg)
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}

	// Должен использовать уровень по умолчанию (info)
	if l == nil {
		t.Fatal("Expected logger to be created with default level")
	}
}

func TestNew_JSONFormat(t *testing.T) {
	cfg := config.LogConfig{
		Level:  "debug",
		Format: "json",
		Output: "stdout",
	}

	l, err := New(cfg)
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}

	if l == nil {
		t.Fatal("Expected logger to be created")
	}
}

func TestDefault(t *testing.T) {
	l := Default()
	if l == nil {
		t.Fatal("Expected default logger to be created")
	}
}

func TestNew_StderrOutput(t *testing.T) {
	cfg := config.LogConfig{
		Level:  "info",
		Format: "text",
		Output: "stderr",
	}

	l, err := New(cfg)
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}

	if l == nil {
		t.Fatal("Expected logger to be created")
	}
}

func TestNew_ErrorLevel(t *testing.T) {
	cfg := config.LogConfig{
		Level:  "error",
		Format: "text",
		Output: "stdout",
	}

	l, err := New(cfg)
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}

	if l == nil {
		t.Fatal("Expected logger to be created")
	}
}

func TestNew_WarnLevel(t *testing.T) {
	cfg := config.LogConfig{
		Level:  "warn",
		Format: "text",
		Output: "stdout",
	}

	l, err := New(cfg)
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}

	if l == nil {
		t.Fatal("Expected logger to be created")
	}
}
