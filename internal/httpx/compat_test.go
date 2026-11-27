package httpx

import (
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

func TestOperationIDFromRequestBodyFallback(t *testing.T) {
	app := fiber.New()
	app.Post("/", func(c *fiber.Ctx) error {
		id, err := OperationIDFromRequest(c, true)
		if err != nil {
			return err
		}
		return c.SendString(id.String())
	})
	op := uuid.New().String()
	req := httptest.NewRequest("POST", "/", strings.NewReader(`{"operationId":"`+op+`"}`))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != 200 {
		t.Fatalf("status=%d", resp.StatusCode)
	}
}

func TestOperationIDFromIdempotencyKeyHeader(t *testing.T) {
	app := fiber.New()
	app.Post("/", func(c *fiber.Ctx) error {
		id, err := OperationIDFromRequest(c, true)
		if err != nil {
			return err
		}
		return c.SendString(id.String())
	})
	op := uuid.New().String()
	req := httptest.NewRequest("POST", "/", strings.NewReader(`{}`))
	req.Header.Set("Idempotency-Key", op)
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != 200 {
		t.Fatalf("status=%d", resp.StatusCode)
	}
}
