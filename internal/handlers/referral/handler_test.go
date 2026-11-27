package referral

import (
	"context"
	"net/http/httptest"
	"strings"
	"testing"

	"backend-hackathon/internal/httpx"
	refmodels "backend-hackathon/internal/models/referral"
	repository "backend-hackathon/internal/repository/repositories"
	svc "backend-hackathon/internal/services/referral"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

type stubService struct {
	listFn       func(context.Context, repository.ActorContext, refmodels.ListQuery, repository.PageRequest) ([]refmodels.View, error)
	transitionFn func(context.Context, repository.ActorContext, uuid.UUID, string, refmodels.TransitionRequest) (*refmodels.View, error)
}

func (s stubService) List(ctx context.Context, a repository.ActorContext, q refmodels.ListQuery, p repository.PageRequest) ([]refmodels.View, error) {
	return s.listFn(ctx, a, q, p)
}
func (s stubService) Transition(ctx context.Context, a repository.ActorContext, id uuid.UUID, ifMatch string, in refmodels.TransitionRequest) (*refmodels.View, error) {
	return s.transitionFn(ctx, a, id, ifMatch, in)
}

func TestTransition409(t *testing.T) {
	h := New(stubService{transitionFn: func(context.Context, repository.ActorContext, uuid.UUID, string, refmodels.TransitionRequest) (*refmodels.View, error) {
		return nil, svc.ErrInvalidTransition
	}})
	app := fiber.New(fiber.Config{ErrorHandler: func(c *fiber.Ctx, err error) error { return httpx.Write(c, err) }})
	app.Use(func(c *fiber.Ctx) error {
		c.Locals("actor", repository.ActorContext{ActorID: uuid.New()})
		return c.Next()
	})
	app.Post("/api/referrals/:referralId/transitions", h.Transition)
	req := httptest.NewRequest("POST", "/api/referrals/"+uuid.New().String()+"/transitions", strings.NewReader(`{"action":"accept"}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("If-Match", "W/\"v1\"")
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != 409 {
		t.Fatalf("status=%d", resp.StatusCode)
	}
}

func TestTransition412(t *testing.T) {
	h := New(stubService{transitionFn: func(context.Context, repository.ActorContext, uuid.UUID, string, refmodels.TransitionRequest) (*refmodels.View, error) {
		return nil, svc.ErrVersionMismatch
	}})
	app := fiber.New(fiber.Config{ErrorHandler: func(c *fiber.Ctx, err error) error { return httpx.Write(c, err) }})
	app.Use(func(c *fiber.Ctx) error {
		c.Locals("actor", repository.ActorContext{ActorID: uuid.New()})
		return c.Next()
	})
	app.Post("/api/referrals/:referralId/transitions", h.Transition)
	req := httptest.NewRequest("POST", "/api/referrals/"+uuid.New().String()+"/transitions", strings.NewReader(`{"action":"accept"}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("If-Match", "W/\"v1\"")
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != 412 {
		t.Fatalf("status=%d", resp.StatusCode)
	}
}
