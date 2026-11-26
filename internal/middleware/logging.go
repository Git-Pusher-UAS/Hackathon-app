package middleware

import (
	"log"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/logger"
)

// Logging usa el middleware oficial conservando el destino y formato del log.
func Logging() fiber.Handler {
	return logger.New(logger.Config{
		Format: "${latency}", // Activa la medición de Start y Stop del middleware.
		LoggerFunc: func(c fiber.Ctx, data *logger.Data, _ *logger.Config) error {
			log.Printf("%s %s %d %dB %s request_id=%s",
				c.Method(), c.OriginalURL(), c.Response().StatusCode(), len(c.Response().Body()),
				data.Stop.Sub(data.Start).Round(time.Microsecond), RequestIDFrom(c.Context()))
			return nil
		},
	})
}
