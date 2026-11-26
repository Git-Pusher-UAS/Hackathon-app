package middleware

import (
	"strings"

	"github.com/gofiber/fiber/v3"
)

// CORS aplica los headers de CORS para el origen configurado.
//
// Si allowedOrigin es "*" se permite cualquier origen. Si es una lista separada
// por comas, solo se permite el origen exacto de la lista.
// El middleware oficial no responde a todo OPTIONS ni incluye estas mismas
// cabeceras en peticiones sin Origin o de origen rechazado.
func CORS(allowedOrigin string) fiber.Handler {
	allowed := strings.TrimSpace(allowedOrigin)
	allowAll := allowed == "*" || allowed == ""

	origins := make(map[string]bool)
	for _, o := range strings.Split(allowed, ",") {
		if o = strings.TrimSpace(o); o != "" && o != "*" {
			origins[o] = true
		}
	}

	return func(c fiber.Ctx) error {
		origin := c.Get("Origin")
		if allowAll {
			c.Set("Access-Control-Allow-Origin", "*")
		} else if origin != "" && origins[origin] {
			c.Set("Access-Control-Allow-Origin", origin)
			c.Append("Vary", "Origin")
		}

		c.Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
		c.Set("Access-Control-Allow-Headers", "Content-Type, Authorization, If-Match, X-Request-ID")
		c.Set("Access-Control-Expose-Headers", HeaderRequestID)
		c.Set("Access-Control-Max-Age", "600")

		if c.Method() == fiber.MethodOptions {
			return c.SendStatus(fiber.StatusNoContent)
		}

		return c.Next()
	}
}
