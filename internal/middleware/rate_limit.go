package middleware

import (
	"sync"
	"time"

	"backend-hackathon/internal/httpx"
	"github.com/gofiber/fiber/v2"
)

type rateBucket struct {
	count int
	reset time.Time
}

type RateLimiter struct {
	mu     sync.Mutex
	m      map[string]rateBucket
	limit  int
	window time.Duration
}

func NewRateLimiter(limit int, window time.Duration) *RateLimiter {
	return &RateLimiter{m: map[string]rateBucket{}, limit: limit, window: window}
}

func (rl *RateLimiter) Allow(key string) bool {
	now := time.Now()
	rl.mu.Lock()
	defer rl.mu.Unlock()
	b, ok := rl.m[key]
	if !ok || now.After(b.reset) {
		rl.m[key] = rateBucket{count: 1, reset: now.Add(rl.window)}
		return true
	}
	if b.count >= rl.limit {
		return false
	}
	b.count++
	rl.m[key] = b
	return true
}

func (rl *RateLimiter) Middleware(keyFn func(*fiber.Ctx) string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		key := keyFn(c)
		if key == "" {
			key = c.IP()
		}
		if !rl.Allow(key) {
			return httpx.E(fiber.StatusTooManyRequests, "TOO_MANY_REQUESTS", "demasiados intentos, intenta más tarde")
		}
		return c.Next()
	}
}
