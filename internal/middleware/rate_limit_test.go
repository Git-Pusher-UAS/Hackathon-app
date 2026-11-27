package middleware

import (
	"net/http/httptest"
	"testing"
	"time"

	"backend-hackathon/internal/httpx"
	"github.com/gofiber/fiber/v2"
)

func TestRateLimiterMiddlewareReturns429(t *testing.T) {
	rl := NewRateLimiter(2, time.Minute)
	app := fiber.New(fiber.Config{ErrorHandler: httpx.Write})
	app.Use(rl.Middleware(func(c *fiber.Ctx) string { return "test-ip" }))
	app.Get("/t", func(c *fiber.Ctx) error { return c.SendStatus(200) })

	for i := 0; i < 2; i++ {
		resp, err := app.Test(httptest.NewRequest("GET", "/t", nil))
		if err != nil {
			t.Fatalf("req %d: %v", i, err)
		}
		if resp.StatusCode != 200 {
			t.Fatalf("req %d status=%d", i, resp.StatusCode)
		}
	}
	resp, err := app.Test(httptest.NewRequest("GET", "/t", nil))
	if err != nil {
		t.Fatalf("req3: %v", err)
	}
	if resp.StatusCode != 429 {
		t.Fatalf("expected 429, got %d", resp.StatusCode)
	}
}
