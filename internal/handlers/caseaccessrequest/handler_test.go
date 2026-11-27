package caseaccessrequest

import (
	"context"
	"net/http/httptest"
	"strings"
	"testing"

	"backend-hackathon/internal/httpx"
	"backend-hackathon/internal/models"
	carmodels "backend-hackathon/internal/models/caseaccessrequest"
	repository "backend-hackathon/internal/repository/repositories"
	svc "backend-hackathon/internal/services/caseaccessrequest"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

type stubService struct {
	createFn func(context.Context, repository.ActorContext, carmodels.CreateRequest) (*carmodels.Created, error)
	listFn   func(context.Context, repository.ActorContext, *models.EstadoSolicitudVinculacion, repository.PageRequest) ([]carmodels.Item, error)
	rejectFn func(context.Context, repository.ActorContext, uuid.UUID, string) error
}

func (s stubService) Create(ctx context.Context, a repository.ActorContext, in carmodels.CreateRequest) (*carmodels.Created, error) {
	return s.createFn(ctx, a, in)
}
func (s stubService) List(ctx context.Context, a repository.ActorContext, st *models.EstadoSolicitudVinculacion, p repository.PageRequest) ([]carmodels.Item, error) {
	return s.listFn(ctx, a, st, p)
}
func (s stubService) Reject(ctx context.Context, a repository.ActorContext, id uuid.UUID, m string) error {
	return s.rejectFn(ctx, a, id, m)
}

func appWith(h *Handler) *fiber.App {
	app := fiber.New(fiber.Config{ErrorHandler: func(c *fiber.Ctx, err error) error { return httpx.Write(c, err) }})
	app.Use(func(c *fiber.Ctx) error {
		c.Locals("actor", repository.ActorContext{ActorID: uuid.New()})
		return c.Next()
	})
	app.Post("/api/case-access-requests", h.Create)
	app.Get("/api/case-access-requests", h.List)
	app.Post("/api/case-access-requests/:id/rejection", h.Reject)
	return app
}

func TestCreate201(t *testing.T) {
	id := uuid.New()
	h := New(stubService{createFn: func(context.Context, repository.ActorContext, carmodels.CreateRequest) (*carmodels.Created, error) {
		return &carmodels.Created{RequestID: id, Estado: models.SolicitudPendiente}, nil
	}})
	req := httptest.NewRequest("POST", "/api/case-access-requests", strings.NewReader(`{"linkToken":"lt_x","motivo":"motivo de prueba suficiente"}`))
	req.Header.Set("Content-Type", "application/json")
	resp, err := appWith(h).Test(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != 201 {
		t.Fatalf("status=%d", resp.StatusCode)
	}
}

func TestCreate410OnExpired(t *testing.T) {
	h := New(stubService{createFn: func(context.Context, repository.ActorContext, carmodels.CreateRequest) (*carmodels.Created, error) {
		return nil, svc.ErrLinkExpired
	}})
	req := httptest.NewRequest("POST", "/api/case-access-requests", strings.NewReader(`{"linkToken":"lt_x","motivo":"motivo de prueba suficiente"}`))
	req.Header.Set("Content-Type", "application/json")
	resp, err := appWith(h).Test(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != 410 {
		t.Fatalf("status=%d", resp.StatusCode)
	}
}

func TestReject204(t *testing.T) {
	h := New(stubService{rejectFn: func(context.Context, repository.ActorContext, uuid.UUID, string) error { return nil }})
	req := httptest.NewRequest("POST", "/api/case-access-requests/"+uuid.New().String()+"/rejection", strings.NewReader(`{"motivo":"motivo de rechazo suficiente"}`))
	req.Header.Set("Content-Type", "application/json")
	resp, err := appWith(h).Test(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != 204 {
		t.Fatalf("status=%d", resp.StatusCode)
	}
}

func TestList200Array(t *testing.T) {
	h := New(stubService{listFn: func(context.Context, repository.ActorContext, *models.EstadoSolicitudVinculacion, repository.PageRequest) ([]carmodels.Item, error) {
		return []carmodels.Item{}, nil
	}})
	req := httptest.NewRequest("GET", "/api/case-access-requests?status=pendiente", nil)
	resp, err := appWith(h).Test(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != 200 {
		t.Fatalf("status=%d", resp.StatusCode)
	}
}
