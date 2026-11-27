// Package user agrupa middleware específico de las rutas /api/users.
package user

import (
	"crypto/sha256"
	"crypto/subtle"
	"strings"

	"backend-hackathon/internal/httpx"
	"backend-hackathon/internal/middleware"

	"github.com/gofiber/fiber/v2"
)

// RequireCSRF exige X-CSRF-Token y lo compara con el hash de la sesión.
// Debe montarse después de AuthMiddleware.
func RequireCSRF(c *fiber.Ctx) error {
	token := strings.TrimSpace(c.Get("X-CSRF-Token"))
	if token == "" {
		return httpx.E(fiber.StatusForbidden, "CSRF_INVALID", "token CSRF inválido")
	}

	expected, ok := middleware.CSRFHash(c)
	if !ok || len(expected) == 0 {
		return httpx.E(fiber.StatusForbidden, "CSRF_INVALID", "token CSRF inválido")
	}

	provided := sha256.Sum256([]byte(token))
	if subtle.ConstantTimeCompare(provided[:], expected) != 1 {
		return httpx.E(fiber.StatusForbidden, "CSRF_INVALID", "token CSRF inválido")
	}

	return c.Next()
}
