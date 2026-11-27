package middleware

import (
	"crypto/sha256"
	"crypto/subtle"
	"strings"
	"time"

	repository "backend-hackathon/internal/repository/repositories"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

const EmployerCookieName = "sai_employer_session"
const EmployerIdleTimeout = 30 * time.Minute

func EmployerAuthMiddleware(uow repository.UnitOfWork) fiber.Handler {
	return func(c *fiber.Ctx) error {
		raw := strings.TrimSpace(c.Cookies(EmployerCookieName))
		if raw == "" {
			return fiber.NewError(fiber.StatusUnauthorized, "sesion invalida")
		}
		parts := strings.SplitN(raw, ".", 2)
		if len(parts) != 2 {
			return fiber.NewError(fiber.StatusUnauthorized, "sesion invalida")
		}
		sid, err := uuid.Parse(parts[0])
		if err != nil {
			return fiber.NewError(fiber.StatusUnauthorized, "sesion invalida")
		}
		token := parts[1]
		repos := uow.Repositories()
		now := time.Now()
		ses, err := repos.EmpleadorSesiones().GetByID(c.Context(), sid)
		if err != nil || ses.RevocadoEn != nil || !ses.VenceAbsoluto.After(now) || !ses.UltimaActividad.Add(EmployerIdleTimeout).After(now) {
			return fiber.NewError(fiber.StatusUnauthorized, "sesion invalida")
		}
		hash := sha256.Sum256([]byte(token))
		if subtle.ConstantTimeCompare(hash[:], ses.TokenHash) != 1 {
			return fiber.NewError(fiber.StatusUnauthorized, "sesion invalida")
		}
		if now.Sub(ses.UltimaActividad) > time.Minute {
			_ = repos.EmpleadorSesiones().Touch(c.Context(), ses.ID, now)
		}
		c.Locals(localEmpleadorID, ses.EmpleadorID)
		c.Locals(localEmpleadorSesionID, ses.ID)
		c.Locals(localEmpleadorCSRFHash, ses.CSRFHash)
		return c.Next()
	}
}

func EmployerCSRF() fiber.Handler {
	return func(c *fiber.Ctx) error {
		switch c.Method() {
		case fiber.MethodGet, fiber.MethodHead, fiber.MethodOptions, fiber.MethodTrace:
			return c.Next()
		}
		hash, ok := EmpleadorCSRFHash(c)
		if !ok || len(hash) == 0 {
			return fiber.NewError(fiber.StatusForbidden, "csrf invalido")
		}
		raw := strings.TrimSpace(c.Get("X-CSRF-Token"))
		if raw == "" {
			return fiber.NewError(fiber.StatusForbidden, "csrf invalido")
		}
		sum := sha256.Sum256([]byte(raw))
		if subtle.ConstantTimeCompare(sum[:], hash) != 1 {
			return fiber.NewError(fiber.StatusForbidden, "csrf invalido")
		}
		return c.Next()
	}
}
