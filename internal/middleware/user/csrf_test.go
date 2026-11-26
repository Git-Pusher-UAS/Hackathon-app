package user

import (
	"crypto/sha256"
	"net/http/httptest"
	"testing"

	"backend-hackathon/internal/httpx"

	"github.com/gofiber/fiber/v2"
)

func TestRequireCSRF(t *testing.T) {
	validToken := "token-csrf-valido"
	validHash := sha256.Sum256([]byte(validToken))

	newApp := func(setHash bool, hash []byte) *fiber.App {
		app := fiber.New(fiber.Config{
			ErrorHandler: func(c *fiber.Ctx, err error) error {
				return httpx.Write(c, err)
			},
		})
		app.Use(func(c *fiber.Ctx) error {
			if setHash {
				c.Locals("csrfHash", hash)
			}
			return c.Next()
		})
		app.Delete("/test", RequireCSRF, func(c *fiber.Ctx) error {
			return c.SendStatus(fiber.StatusNoContent)
		})
		return app
	}

	t.Run("rechaza cabecera faltante", func(t *testing.T) {
		resp, err := newApp(true, validHash[:]).Test(httptest.NewRequest("DELETE", "/test", nil))
		if err != nil {
			t.Fatalf("app.Test: %v", err)
		}
		if resp.StatusCode != fiber.StatusForbidden {
			t.Fatalf("status = %d, esperado 403", resp.StatusCode)
		}
	})

	t.Run("rechaza ausencia de hash en contexto", func(t *testing.T) {
		req := httptest.NewRequest("DELETE", "/test", nil)
		req.Header.Set("X-CSRF-Token", validToken)
		resp, err := newApp(false, nil).Test(req)
		if err != nil {
			t.Fatalf("app.Test: %v", err)
		}
		if resp.StatusCode != fiber.StatusForbidden {
			t.Fatalf("status = %d, esperado 403", resp.StatusCode)
		}
	})

	t.Run("rechaza token distinto", func(t *testing.T) {
		req := httptest.NewRequest("DELETE", "/test", nil)
		req.Header.Set("X-CSRF-Token", "otro-token")
		resp, err := newApp(true, validHash[:]).Test(req)
		if err != nil {
			t.Fatalf("app.Test: %v", err)
		}
		if resp.StatusCode != fiber.StatusForbidden {
			t.Fatalf("status = %d, esperado 403", resp.StatusCode)
		}
	})

	t.Run("permite token válido", func(t *testing.T) {
		req := httptest.NewRequest("DELETE", "/test", nil)
		req.Header.Set("X-CSRF-Token", validToken)
		resp, err := newApp(true, validHash[:]).Test(req)
		if err != nil {
			t.Fatalf("app.Test: %v", err)
		}
		if resp.StatusCode != fiber.StatusNoContent {
			t.Fatalf("status = %d, esperado 204", resp.StatusCode)
		}
	})
}

func TestNoStore(t *testing.T) {
	app := fiber.New()
	app.Use(NoStore)
	app.Get("/test", func(c *fiber.Ctx) error {
		return c.SendStatus(fiber.StatusOK)
	})

	resp, err := app.Test(httptest.NewRequest("GET", "/test", nil))
	if err != nil {
		t.Fatalf("app.Test: %v", err)
	}
	if resp.Header.Get(fiber.HeaderCacheControl) != "no-store" || resp.Header.Get(fiber.HeaderPragma) != "no-cache" {
		t.Fatalf("headers inesperados: %v", resp.Header)
	}
}
