package user

import "github.com/gofiber/fiber/v2"

// NoStore impide que respuestas con datos de identidad se guarden en caché.
func NoStore(c *fiber.Ctx) error {
	c.Set(fiber.HeaderCacheControl, "no-store")
	c.Set(fiber.HeaderPragma, "no-cache")
	return c.Next()
}
