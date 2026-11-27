package user

import (
	"slices"

	"backend-hackathon/internal/httpx"
	"backend-hackathon/internal/middleware"

	"github.com/gofiber/fiber/v2"
)

const (
	// PermAdmin administra cuentas y sesiones.
	PermAdmin = "admin"
	// PermReadAttention lista solo al personal de atención activo (Coordinación, D-04).
	PermReadAttention = "users.read_attention"
)

// DescripcionReadAttention la usa el seed.
const DescripcionReadAttention = "Listar al personal de atención activo para asignar expedientes"

// RequireAny deja pasar si el token tiene alguno de los permisos.
func RequireAny(codigos ...string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		perms := middleware.Permisos(c)
		for _, p := range codigos {
			if slices.Contains(perms, p) {
				return c.Next()
			}
		}
		return httpx.E(fiber.StatusForbidden, "forbidden", "permiso insuficiente")
	}
}
