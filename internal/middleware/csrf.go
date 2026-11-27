package middleware

import (
	"errors"
	"log"

	"backend-hackathon/internal/auth"
	repository "backend-hackathon/internal/repository/repositories"

	"github.com/gofiber/fiber/v2"
)

const headerCSRF = "X-CSRF-Token"

// CSRF exige el header X-CSRF-Token en las peticiones que modifican datos.
// Debe ir DESPU\u00c9S de AuthMiddleware, porque necesita el SessionID en el contexto.
func CSRF(uow repository.UnitOfWork) fiber.Handler {
	return func(c *fiber.Ctx) error {
		switch c.Method() {
		case fiber.MethodGet, fiber.MethodHead, fiber.MethodOptions:
			return c.Next()
		}

		sessionID, ok := SessionID(c)
		if !ok {
			return fiber.NewError(fiber.StatusUnauthorized, auth.ErrTokenInvalido.Error())
		}

		token := c.Get(headerCSRF)
		if token == "" {
			return fiber.NewError(fiber.StatusForbidden, "token csrf requerido")
		}

		sesion, err := uow.Repositories().Sesiones().GetByID(c.Context(), sessionID)
		if err != nil {
			if errors.Is(err, repository.ErrNotFound) {
				return fiber.NewError(fiber.StatusUnauthorized, auth.ErrSesionInvalida.Error())
			}
			log.Printf("csrf: cargar sesi\u00f3n: %v", err)
			return fiber.NewError(fiber.StatusInternalServerError, "error interno")
		}

		if !auth.CSRFValido(sesion.CSRFHash, token) {
			return fiber.NewError(fiber.StatusForbidden, "token csrf inv\u00e1lido")
		}

		return c.Next()
	}
}
