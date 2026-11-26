package middleware

import (
	"log"
	"runtime/debug"

	"backend-hackathon/internal/httpx"

	"github.com/gofiber/fiber/v3"
	fiberrecover "github.com/gofiber/fiber/v3/middleware/recover"
)

// Recover convierte un panic en un 500 JSON mediante el middleware de Fiber.
func Recover() fiber.Handler {
	return fiberrecover.New(fiberrecover.Config{
		EnableStackTrace: true,
		StackTraceHandler: func(c fiber.Ctx, rec any) {
			log.Printf("panic en %s %s (request_id=%s): %v\n%s",
				c.Method(), c.Path(), RequestIDFrom(c.Context()), rec, debug.Stack())
		},
		PanicHandler: func(c fiber.Ctx, _ any) error {
			return httpx.Error(c, fiber.StatusInternalServerError, "internal_error", "error interno del servidor")
		},
	})
}
