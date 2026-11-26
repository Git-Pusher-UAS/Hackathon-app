package handlers

import (
	"time"

	"github.com/gofiber/fiber/v2"
)

// Healthcheck es el liveness probe. No toca la base de datos.
func Healthcheck(c *fiber.Ctx) error {
	return c.JSON(fiber.Map{
		"status": "OK",
		"time":   time.Now().Format(time.RFC3339),
	})
}
