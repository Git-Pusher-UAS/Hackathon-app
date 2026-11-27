package referral

import (
	"backend-hackathon/internal/middleware"

	"github.com/gofiber/fiber/v2"
)

const (
	PermRead       = "referral.read"
	PermTransition = "referral.transition"
)

func Require(codigo string) fiber.Handler { return middleware.RequirePermiso(codigo) }
