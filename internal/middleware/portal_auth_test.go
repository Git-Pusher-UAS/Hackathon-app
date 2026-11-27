package middleware

import (
	"context"
	"net/http/httptest"
	"testing"

	repository "backend-hackathon/internal/repository/repositories"

	"github.com/gofiber/fiber/v2"
)

type noopUoW struct{ repository.UnitOfWork }

func (noopUoW) Do(ctx context.Context, _ repository.ActorContext, fn func(context.Context, repository.Repositories) error) error {
	return fn(ctx, nil)
}
func (noopUoW) DoSystem(ctx context.Context, fn func(context.Context, repository.Repositories) error) error {
	return fn(ctx, nil)
}
func (noopUoW) DoIndependent(ctx context.Context, fn func(context.Context, repository.Repositories) error) error {
	return fn(ctx, nil)
}
func (noopUoW) Repositories() repository.Repositories { return nil }

func TestPortalAuthRejectsStaffCookie(t *testing.T) {
	app := fiber.New()
	app.Use(PortalAuthMiddleware(noopUoW{}))
	app.Get("/ok", func(c *fiber.Ctx) error { return c.SendStatus(200) })
	req := httptest.NewRequest("GET", "/ok", nil)
	req.Header.Set("Cookie", "sai_session=stafftoken")
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != 401 {
		t.Fatalf("status=%d", resp.StatusCode)
	}
}

func TestPortalAuthRejectsMissingCookie(t *testing.T) {
	app := fiber.New()
	app.Use(PortalAuthMiddleware(noopUoW{}))
	app.Get("/ok", func(c *fiber.Ctx) error { return c.SendStatus(200) })
	resp, err := app.Test(httptest.NewRequest("GET", "/ok", nil))
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != 401 {
		t.Fatalf("status=%d", resp.StatusCode)
	}
}
