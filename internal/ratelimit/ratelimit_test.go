package ratelimit

import (
	"testing"
	"time"
)

func TestNewRateLimiter(t *testing.T) {
	rl := NewRateLimiter(60, 10)

	if rl.rps != 1.0 {
		t.Errorf("Expected rps to be 1.0, got %f", rl.rps)
	}

	if rl.burst != 10 {
		t.Errorf("Expected burst to be 10, got %d", rl.burst)
	}
}

func TestRateLimiter_Allow(t *testing.T) {
	rl := NewRateLimiter(60, 5) // 1 запрос в секунду, burst 5

	chatID := int64(12345)

	// Первые 5 запросов должны пройти (burst)
	for i := 0; i < 5; i++ {
		if !rl.Allow(chatID) {
			t.Errorf("Request %d should be allowed", i)
		}
	}

	// 6-й запрос должен быть отклонён
	if rl.Allow(chatID) {
		t.Error("Request 6 should be rate limited")
	}
}

func TestRateLimiter_DifferentUsers(t *testing.T) {
	rl := NewRateLimiter(60, 2)

	// Пользователь 1 исчерпывает лимит
	rl.Allow(1)
	rl.Allow(1)

	// Пользователь 2 должен иметь свой лимит
	if !rl.Allow(2) {
		t.Error("User 2 should be allowed")
	}
}

func TestRateLimiter_Reset(t *testing.T) {
	rl := NewRateLimiter(60, 2)
	chatID := int64(12345)

	// Исчерпываем лимит
	rl.Allow(chatID)
	rl.Allow(chatID)

	// Сбрасываем
	rl.Reset(chatID)

	// Теперь снова можно делать запросы
	if !rl.Allow(chatID) {
		t.Error("Should be allowed after reset")
	}
}

func TestRateLimiter_GetActiveUsers(t *testing.T) {
	rl := NewRateLimiter(60, 10)

	if rl.GetActiveUsers() != 0 {
		t.Errorf("Expected 0 active users, got %d", rl.GetActiveUsers())
	}

	rl.Allow(1)
	rl.Allow(2)
	rl.Allow(3)

	if rl.GetActiveUsers() != 3 {
		t.Errorf("Expected 3 active users, got %d", rl.GetActiveUsers())
	}
}

func TestRateLimiter_Wait(t *testing.T) {
	rl := NewRateLimiter(600, 1) // 10 запросов в секунду

	chatID := int64(12345)

	// Первый запрос проходит сразу
	if !rl.Allow(chatID) {
		t.Error("First request should be allowed")
	}

	// Ждём возможности следующего запроса
	err := rl.Wait(chatID)
	if err != nil {
		t.Errorf("Wait returned error: %v", err)
	}
}

func TestRateLimiter_Concurrent(t *testing.T) {
	rl := NewRateLimiter(60, 10)

	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func(id int64) {
			rl.Allow(id)
			done <- true
		}(int64(i))
	}

	for i := 0; i < 10; i++ {
		<-done
	}
}

func TestRateLimiter_TokenBucketRefill(t *testing.T) {
	rl := NewRateLimiter(60, 2) // 1 запрос в секунду, burst 2
	chatID := int64(12345)

	// Исчерпываем burst
	rl.Allow(chatID)
	rl.Allow(chatID)

	// Ждём пополнения токенов
	time.Sleep(2 * time.Second)

	// Теперь должен пройти ещё один запрос
	if !rl.Allow(chatID) {
		t.Error("Should be allowed after token refill")
	}
}
