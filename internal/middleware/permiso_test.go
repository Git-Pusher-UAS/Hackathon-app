package middleware

import (
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
)

func TestRequirePermiso(t *testing.T) {
	newApp := func(perms []string) *fiber.App {
		app := fiber.New()
		app.Use(func(c *fiber.Ctx) error {
			if perms != nil {
				c.Locals("permisos", perms)
			}
			return c.Next()
		})
		app.Get("/admin", RequirePermiso("admin"), func(c *fiber.Ctx) error {
			return c.SendStatus(fiber.StatusOK)
		})
		return app
	}

	t.Run("rechaza cuando falta el permiso", func(t *testing.T) {
		resp, err := newApp([]string{"case_files.read"}).Test(httptest.NewRequest("GET", "/admin", nil))
		if err != nil {
			t.Fatalf("app.Test: %v", err)
		}
		if resp.StatusCode != fiber.StatusForbidden {
			t.Fatalf("status = %d, esperado 403", resp.StatusCode)
		}
	})

	t.Run("permite cuando el permiso está presente", func(t *testing.T) {
		resp, err := newApp([]string{"case_files.read", "admin"}).Test(httptest.NewRequest("GET", "/admin", nil))
		if err != nil {
			t.Fatalf("app.Test: %v", err)
		}
		if resp.StatusCode != fiber.StatusOK {
			t.Fatalf("status = %d, esperado 200", resp.StatusCode)
		}
	})

	t.Run("admin bypass otorga acceso a otro permiso", func(t *testing.T) {
		app := fiber.New()
		app.Use(func(c *fiber.Ctx) error {
			c.Locals("permisos", []string{"admin"})
			return c.Next()
		})
		app.Get("/courses", RequirePermiso("courses.manage"), func(c *fiber.Ctx) error { return c.SendStatus(fiber.StatusOK) })
		resp, err := app.Test(httptest.NewRequest("GET", "/courses", nil))
		if err != nil {
			t.Fatalf("app.Test: %v", err)
		}
		if resp.StatusCode != fiber.StatusOK {
			t.Fatalf("status = %d, esperado 200", resp.StatusCode)
		}
	})
}
