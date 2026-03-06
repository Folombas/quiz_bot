package ratelimit

import (
	"context"
	"sync"
	"time"

	"golang.org/x/time/rate"
)

// RateLimiter — ограничитель запросов
type RateLimiter struct {
	mu       sync.RWMutex
	limiters map[int64]*rate.Limiter
	rps      float64
	burst    int
}

// NewRateLimiter создаёт новый ограничитель
func NewRateLimiter(requestsPerMin int, burstSize int) *RateLimiter {
	rps := float64(requestsPerMin) / 60.0
	return &RateLimiter{
		limiters: make(map[int64]*rate.Limiter),
		rps:      rps,
		burst:    burstSize,
	}
}

// getLimiter получает или создаёт ограничитель для пользователя
func (rl *RateLimiter) getLimiter(chatID int64) *rate.Limiter {
	rl.mu.RLock()
	limiter, exists := rl.limiters[chatID]
	rl.mu.RUnlock()

	if exists {
		return limiter
	}

	rl.mu.Lock()
	defer rl.mu.Unlock()

	// Проверяем ещё раз после получения write lock
	if limiter, exists = rl.limiters[chatID]; exists {
		return limiter
	}

	limiter = rate.NewLimiter(rate.Limit(rl.rps), rl.burst)
	rl.limiters[chatID] = limiter
	return limiter
}

// Allow проверяет, можно ли выполнить запрос
func (rl *RateLimiter) Allow(chatID int64) bool {
	limiter := rl.getLimiter(chatID)
	return limiter.Allow()
}

// Wait ждёт возможности выполнить запрос
func (rl *RateLimiter) Wait(chatID int64) error {
	limiter := rl.getLimiter(chatID)
	return limiter.Wait(context.Background())
}

// Reset сбрасывает ограничитель для пользователя
func (rl *RateLimiter) Reset(chatID int64) {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	delete(rl.limiters, chatID)
}

// Cleanup удаляет старые ограничители
func (rl *RateLimiter) Cleanup(maxAge time.Duration) {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	// В данной реализации не отслеживаем время последней активности
	// Можно добавить map[chatID]time.Time для tracking
}

// GetActiveUsers возвращает количество активных ограничителей
func (rl *RateLimiter) GetActiveUsers() int {
	rl.mu.RLock()
	defer rl.mu.RUnlock()
	return len(rl.limiters)
}
