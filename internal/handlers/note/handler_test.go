package note

import (
	"context"
	"net/http/httptest"
	"strings"
	"testing"

	"backend-hackathon/internal/httpx"
	nmodels "backend-hackathon/internal/models/note"
	repository "backend-hackathon/internal/repository/repositories"
	svc "backend-hackathon/internal/services/note"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

type stubService struct {
	syncFn func(context.Context, repository.ActorContext, nmodels.SyncRequest) ([]nmodels.SyncResult, error)
}

func (s stubService) Sync(ctx context.Context, a repository.ActorContext, in nmodels.SyncRequest) ([]nmodels.SyncResult, error) {
	return s.syncFn(ctx, a, in)
}

func TestSync200Array(t *testing.T) {
	op := uuid.New().String()
	h := New(stubService{syncFn: func(_ context.Context, _ repository.ActorContext, _ nmodels.SyncRequest) ([]nmodels.SyncResult, error) {
		return []nmodels.SyncResult{{ID: op, Result: nmodels.ResultApplied}}, nil
	}})
	app := fiber.New(fiber.Config{ErrorHandler: func(c *fiber.Ctx, err error) error { return httpx.Write(c, err) }})
	app.Use(func(c *fiber.Ctx) error {
		c.Locals("actor", repository.ActorContext{ActorID: uuid.New()})
		return c.Next()
	})
	app.Post("/api/notes/sync", h.Sync)
	body := `{"notes":[{"operationId":"` + op + `","recordId":"` + uuid.New().String() + `","departmentId":"` + uuid.New().String() + `","texto":"hola","ocurridoEn":"2026-11-27T10:00:00Z"}]}`
	req := httptest.NewRequest("POST", "/api/notes/sync", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != 200 {
		t.Fatalf("status=%d", resp.StatusCode)
	}
}

func TestSync422OnBatchValidation(t *testing.T) {
	h := New(stubService{syncFn: func(context.Context, repository.ActorContext, nmodels.SyncRequest) ([]nmodels.SyncResult, error) {
		return nil, &svc.ValidationError{Msg: "notes debe tener entre 1 y 20 elementos"}
	}})
	app := fiber.New(fiber.Config{ErrorHandler: func(c *fiber.Ctx, err error) error { return httpx.Write(c, err) }})
	app.Use(func(c *fiber.Ctx) error {
		c.Locals("actor", repository.ActorContext{ActorID: uuid.New()})
		return c.Next()
	})
	app.Post("/api/notes/sync", h.Sync)
	req := httptest.NewRequest("POST", "/api/notes/sync", strings.NewReader(`{"notes":[]}`))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != 422 {
		t.Fatalf("status=%d", resp.StatusCode)
	}
}
