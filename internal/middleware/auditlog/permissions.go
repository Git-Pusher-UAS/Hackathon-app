package auditlog

import (
	"backend-hackathon/internal/middleware"

	"github.com/gofiber/fiber/v2"
)

const PermRead = "auditlog.read"

func Require() fiber.Handler { return middleware.RequirePermiso(PermRead) }
