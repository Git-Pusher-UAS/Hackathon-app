package middleware

import (
	"slices"

	"github.com/gofiber/fiber/v2"
)

// RequirePermiso exige que el token autenticado incluya el permiso indicado.
// El permiso "admin" otorga acceso total. Debe montarse después de AuthMiddleware.
func RequirePermiso(codigo string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		if hasPermiso(c, codigo) {
			return c.Next()
		}
		return fiber.NewError(fiber.StatusForbidden, "permiso insuficiente")
	}
}

func hasPermiso(c *fiber.Ctx, codigo string) bool {
	perms := Permisos(c)
	return slices.Contains(perms, "admin") || slices.Contains(perms, codigo)
}
