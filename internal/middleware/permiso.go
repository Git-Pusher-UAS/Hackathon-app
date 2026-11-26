package middleware

import (
	"slices"

	"github.com/gofiber/fiber/v2"
)

// RequirePermiso exige que el token autenticado incluya el permiso indicado.
// Debe montarse después de AuthMiddleware.
func RequirePermiso(codigo string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		if !slices.Contains(Permisos(c), codigo) {
			return fiber.NewError(fiber.StatusForbidden, "permiso insuficiente")
		}
		return c.Next()
	}
}
