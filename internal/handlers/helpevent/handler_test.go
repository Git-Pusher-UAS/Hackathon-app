package helpevent

import (
	"context"
	"net/http/httptest"
	"strings"
	"testing"

	"backend-hackathon/internal/httpx"
	hemodels "backend-hackathon/internal/models/helpevent"
	repository "backend-hackathon/internal/repository/repositories"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

type stubService struct {
	recordFn func(context.Context, repository.ActorContext, uuid.UUID, hemodels.CreateRequest) error
}

func (s stubService) Record(ctx context.Context, actor repository.ActorContext, opID uuid.UUID, in hemodels.CreateRequest) error {
	return s.recordFn(ctx, actor, opID, in)
}

func TestHelpEventCreate204(t *testing.T) {
	op := uuid.New()
	var gotOp uuid.UUID
	app := fiber.New(fiber.Config{ErrorHandler: func(c *fiber.Ctx, err error) error { return httpx.Write(c, err) }})
	h := New(stubService{recordFn: func(_ context.Context, _ repository.ActorContext, id uuid.UUID, _ hemodels.CreateRequest) error {
		gotOp = id
		return nil
	}})
	app.Post("/api/help-events", func(c *fiber.Ctx) error {
		c.Locals("actor", repository.ActorContext{ActorID: uuid.New()})
		return c.Next()
	}, h.Create)

	req := httptest.NewRequest("POST", "/api/help-events", strings.NewReader(`{"contactId":"`+uuid.New().String()+`","action":"dialer_opened","occurredAt":"2026-11-27T10:00:00Z","operationId":"`+op.String()+`"}`))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != 204 {
		t.Fatalf("status=%d", resp.StatusCode)
	}
	if gotOp != op {
		t.Fatalf("op=%s want=%s", gotOp, op)
	}
}

func TestHelpEventCreateInvalidOperationID(t *testing.T) {
	app := fiber.New(fiber.Config{ErrorHandler: func(c *fiber.Ctx, err error) error { return httpx.Write(c, err) }})
	h := New(stubService{recordFn: func(context.Context, repository.ActorContext, uuid.UUID, hemodels.CreateRequest) error { return nil }})
	app.Post("/api/help-events", func(c *fiber.Ctx) error {
		c.Locals("actor", repository.ActorContext{ActorID: uuid.New()})
		return c.Next()
	}, h.Create)
	req := httptest.NewRequest("POST", "/api/help-events", strings.NewReader(`{"action":"dialer_opened","occurredAt":"2026-11-27T10:00:00Z","operationId":"bad"}`))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != 422 {
		t.Fatalf("status=%d", resp.StatusCode)
	}
}
