package middleware

import (
	"slices"

	"github.com/gofiber/fiber/v2"
)

// RequirePermiso exige que el token autenticado incluya el permiso indicado.
// El permiso "admin" otorga acceso total. Debe montarse después de AuthMiddleware.
func RequirePermiso(codigo string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		perms := Permisos(c)
		if slices.Contains(perms, "admin") || slices.Contains(perms, codigo) {
			return c.Next()
		}
		return fiber.NewError(fiber.StatusForbidden, "permiso insuficiente")
	}
}
