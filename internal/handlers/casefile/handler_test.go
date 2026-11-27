package casefile

import (
	"context"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"backend-hackathon/internal/httpx"
	casemodels "backend-hackathon/internal/models/casefile"
	repository "backend-hackathon/internal/repository/repositories"
	svc "backend-hackathon/internal/services/casefile"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

type stubService struct{ svc.Service }

func (s stubService) DuplicateCheck(context.Context, repository.ActorContext, string) (*casemodels.DuplicateCheckResponse, error) {
	return nil, nil
}
func (s stubService) Create(_ context.Context, _ repository.ActorContext, _ uuid.UUID, _ casemodels.CreateRequest) (*casemodels.Summary, error) {
	return s.Service.(interface {
		create() (*casemodels.Summary, error)
	}).create()
}

type createStub struct {
	stubService
	fn func() (*casemodels.Summary, error)
}

func (c createStub) create() (*casemodels.Summary, error) { return c.fn() }
func (c createStub) Create(_ context.Context, _ repository.ActorContext, _ uuid.UUID, _ casemodels.CreateRequest) (*casemodels.Summary, error) {
	return c.fn()
}

func TestCreate201HasLocation(t *testing.T) {
	id := uuid.New()
	h := New(createStub{fn: func() (*casemodels.Summary, error) { return &casemodels.Summary{ID: id, VersionTag: "v1"}, nil }})
	app := fiber.New(fiber.Config{ErrorHandler: func(c *fiber.Ctx, err error) error { return httpx.Write(c, err) }})
	app.Use(func(c *fiber.Ctx) error {
		c.Locals("actor", repository.ActorContext{ActorID: uuid.New()})
		return c.Next()
	})
	app.Post("/api/case-files", h.Create)
	req := httptest.NewRequest("POST", "/api/case-files", strings.NewReader(`{}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Operation-Id", uuid.New().String())
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != 201 {
		t.Fatalf("status=%d", resp.StatusCode)
	}
	if loc := resp.Header.Get("Location"); loc != "/api/case-files/"+id.String() {
		t.Fatalf("location=%q", loc)
	}
}

func TestCreate409PersonExists(t *testing.T) {
	link := "lt_abc"
	exp := time.Now().Add(time.Hour)
	h := New(createStub{fn: func() (*casemodels.Summary, error) {
		return nil, &svc.DuplicatePersonError{LinkToken: &link, LinkExpiresAt: &exp}
	}})
	app := fiber.New(fiber.Config{ErrorHandler: func(c *fiber.Ctx, err error) error { return httpx.Write(c, err) }})
	app.Use(func(c *fiber.Ctx) error {
		c.Locals("actor", repository.ActorContext{ActorID: uuid.New()})
		return c.Next()
	})
	app.Post("/api/case-files", h.Create)
	req := httptest.NewRequest("POST", "/api/case-files", strings.NewReader(`{}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Operation-Id", uuid.New().String())
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != 409 {
		t.Fatalf("status=%d", resp.StatusCode)
	}
}
