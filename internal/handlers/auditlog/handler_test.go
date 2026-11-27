package auditlog

import (
	"context"
	"net/http/httptest"
	"testing"
	"time"

	"backend-hackathon/internal/httpx"
	almodels "backend-hackathon/internal/models/auditlog"
	repository "backend-hackathon/internal/repository/repositories"
	svc "backend-hackathon/internal/services/auditlog"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

type stubService struct {
	logsFn func(context.Context, repository.ActorContext, svc.Query, repository.PageRequest) ([]almodels.Entry, error)
}

func (s stubService) Logs(ctx context.Context, a repository.ActorContext, q svc.Query, p repository.PageRequest) ([]almodels.Entry, error) {
	return s.logsFn(ctx, a, q, p)
}

func TestList200Array(t *testing.T) {
	h := New(stubService{logsFn: func(context.Context, repository.ActorContext, svc.Query, repository.PageRequest) ([]almodels.Entry, error) {
		return []almodels.Entry{{ID: uuid.New(), Fecha: time.Now(), Accion: "casefile.read"}}, nil
	}})
	app := fiber.New(fiber.Config{ErrorHandler: func(c *fiber.Ctx, err error) error { return httpx.Write(c, err) }})
	app.Use(func(c *fiber.Ctx) error {
		c.Locals("actor", repository.ActorContext{ActorID: uuid.New()})
		return c.Next()
	})
	app.Get("/api/audit-logs", h.List)
	resp, err := app.Test(httptest.NewRequest("GET", "/api/audit-logs?action=casefile.read", nil))
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != 200 {
		t.Fatalf("status=%d", resp.StatusCode)
	}
}
