package httpapi

import (
	"sync"
	"time"
)

type loginAttempt struct {
	count   int
	resetAt time.Time
}

type loginLimiter struct {
	mu       sync.Mutex
	entries  map[string]loginAttempt
	limit    int
	interval time.Duration
}

func newLoginLimiter(limit int, interval time.Duration) *loginLimiter {
	return &loginLimiter{entries: make(map[string]loginAttempt), limit: limit, interval: interval}
}

func (limiter *loginLimiter) Allow(key string) bool {
	limiter.mu.Lock()
	defer limiter.mu.Unlock()

	now := time.Now()
	for address, attempt := range limiter.entries {
		if !now.Before(attempt.resetAt) {
			delete(limiter.entries, address)
		}
	}

	attempt := limiter.entries[key]
	if !now.Before(attempt.resetAt) {
		attempt = loginAttempt{resetAt: now.Add(limiter.interval)}
	}
	if attempt.count >= limiter.limit {
		limiter.entries[key] = attempt
		return false
	}
	attempt.count++
	limiter.entries[key] = attempt
	return true
}
